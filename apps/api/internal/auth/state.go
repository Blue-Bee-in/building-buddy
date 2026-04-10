package auth

import (
	"net/http"
	"strings"
	"time"

	ab "github.com/volatiletech/authboss/v3"
	"github.com/golang-jwt/jwt/v5"
)

// jwtState implements authboss.ClientState.
// Storing the full data map lets WriteState read back existing state
// so it can correctly merge events on top of it.
type jwtState struct {
	data map[string]string
}

func (j *jwtState) Get(key string) (string, bool) {
	v, ok := j.data[key]
	return v, ok
}

// JWTStateRW implements authboss.ClientStateReadWriter using short-lived
// HS256 JWTs carried in the Authorization: Bearer header.
//
// This replaces AuthBoss's cookie/session state entirely — the JWT IS the
// session for mobile clients.
type JWTStateRW struct {
	secret []byte
	expiry time.Duration
}

func NewJWTStateRW(secret []byte, expiry time.Duration) *JWTStateRW {
	return &JWTStateRW{secret: secret, expiry: expiry}
}

// ReadState parses the Bearer token from the Authorization header and
// returns the embedded AuthBoss state claims.  An absent or invalid token
// returns an empty (not-logged-in) state — not an error.
func (rw *JWTStateRW) ReadState(r *http.Request) (ab.ClientState, error) {
	empty := &jwtState{data: map[string]string{}}

	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return empty, nil
	}

	token, err := jwt.Parse(
		strings.TrimPrefix(header, "Bearer "),
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return rw.secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
	)
	if err != nil || !token.Valid {
		return empty, nil // expired / tampered → treat as logged-out
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return empty, nil
	}

	state := &jwtState{data: make(map[string]string, len(claims))}
	for k, v := range claims {
		if s, ok := v.(string); ok {
			state.data[k] = s
		}
	}
	return state, nil
}

// WriteState applies the AuthBoss state events, re-signs the JWT and sets
// it in the X-Auth-Token response header so the client can store it.
// If events leave the state empty (logout), the header is set to an empty string.
func (rw *JWTStateRW) WriteState(w http.ResponseWriter, cs ab.ClientState, events []ab.ClientStateEvent) error {
	if len(events) == 0 {
		return nil // nothing changed — client keeps its current token
	}

	// Copy current state
	data := map[string]string{}
	if jcs, ok := cs.(*jwtState); ok {
		for k, v := range jcs.data {
			data[k] = v
		}
	}

	// Apply events
	for _, e := range events {
		switch e.Kind {
		case ab.ClientStateEventPut:
			data[e.Key] = e.Value
		case ab.ClientStateEventDel:
			delete(data, e.Key)
		case ab.ClientStateEventDelAll:
			// Key field is a comma-separated whitelist of keys to KEEP
			if e.Key == "" {
				data = map[string]string{}
			} else {
				keep := make(map[string]bool)
				for _, k := range strings.Split(e.Key, ",") {
					keep[strings.TrimSpace(k)] = true
				}
				for k := range data {
					if !keep[k] {
						delete(data, k)
					}
				}
			}
		}
	}

	if len(data) == 0 {
		w.Header().Set("X-Auth-Token", "")
		return nil
	}

	claims := jwt.MapClaims{
		"iat": jwt.NewNumericDate(time.Now()),
		"exp": jwt.NewNumericDate(time.Now().Add(rw.expiry)),
	}
	for k, v := range data {
		claims[k] = v
	}

	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString(rw.secret)
	if err != nil {
		return err
	}
	w.Header().Set("X-Auth-Token", tokenStr)
	return nil
}

// noopCookieState is a no-op ClientStateReadWriter for the cookie slot —
// mobile apps don't use cookies.
type noopCookieState struct{}

func (*noopCookieState) ReadState(_ *http.Request) (ab.ClientState, error) {
	return &jwtState{data: map[string]string{}}, nil
}
func (*noopCookieState) WriteState(_ http.ResponseWriter, _ ab.ClientState, _ []ab.ClientStateEvent) error {
	return nil
}
