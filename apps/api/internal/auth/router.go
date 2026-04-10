package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	ab "github.com/volatiletech/authboss/v3"
	_ "github.com/volatiletech/authboss/v3/auth"    // password login
	_ "github.com/volatiletech/authboss/v3/lock"    // brute-force lockout
	_ "github.com/volatiletech/authboss/v3/logout"  // logout
	_ "github.com/volatiletech/authboss/v3/otp"     // OTP-based login
	_ "github.com/volatiletech/authboss/v3/recover" // forgot-password recovery
	_ "github.com/volatiletech/authboss/v3/register" // sign-up
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appjwt "github.com/Blue-Bee-in/building-buddy/pkg/jwt"
)

// Setup initialises AuthBoss with all required adapters and returns an
// http.Handler that serves both the AuthBoss-managed routes and our custom
// ones (phone OTP verify, refresh, invite accept).
func Setup(db *pgxpool.Pool) (http.Handler, error) {
	storer := NewPGStorer(db)
	secret := []byte(os.Getenv("JWT_SECRET"))
	jwtRW := NewJWTStateRW(secret, 15*time.Minute)

	a := ab.New()

	a.Config.Paths.Mount = "/api/v1/auth"
	a.Config.Paths.RootURL = os.Getenv("API_URL")

	a.Config.Storage.Server = storer
	a.Config.Storage.SessionState = jwtRW
	a.Config.Storage.CookieState = &noopCookieState{}

	// Wire JSON adapters
	renderer := JSONRenderer{}
	a.Config.Core.ViewRenderer = renderer
	a.Config.Core.MailRenderer = renderer
	a.Config.Core.BodyReader = JSONBodyReader{}
	a.Config.Core.Responder = JSONResponder{}
	a.Config.Core.Redirector = JSONRedirector{}
	a.Config.Core.Mailer = SMSMailer{}
	a.Config.Core.Logger = SlogLogger{}

	// Give AuthBoss a chi-backed router to register its own routes on
	cr := chi.NewRouter()
	a.Config.Core.Router = &chiABRouter{mux: cr}

	// ── Event hooks ────────────────────────────────────────────────────────────

	// After successful login (password or OTP): issue refresh token + full response
	a.Events.After(ab.EventAuth, afterAuth(db, storer, a))

	if err := a.Init(); err != nil {
		return nil, err
	}

	// ── Mount everything on a single chi router ────────────────────────────────
	r := chi.NewRouter()
	r.Use(a.LoadClientStateMiddleware)

	// AuthBoss routes: /login, /logout, /register, /recover, /otp/login …
	r.Mount("/", cr)

	// Custom routes
	r.Post("/otp/send", handlePhoneOTPSend(db))
	r.Post("/otp/verify", handlePhoneOTPVerify(db, storer, a))
	r.Post("/refresh", handleRefresh(db, storer))
	r.Group(func(r chi.Router) {
		r.Use(ab.Middleware2(a, ab.RequireFullAuth, ab.RespondUnauthorized))
		r.Put("/profile", handleProfile(storer, a))
		r.Post("/invite/accept", handleInviteAccept(db, a))
	})

	return r, nil
}

// ── After(EventAuth) hook ─────────────────────────────────────────────────────

type loginResponse struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
	ExpiresIn    int              `json:"expires_in"`
	User         UserDTO          `json:"user"`
	Memberships  []FlatMembership `json:"memberships"`
}

func afterAuth(db *pgxpool.Pool, storer *PGStorer, a *ab.Authboss) func(http.ResponseWriter, *http.Request, bool) (bool, error) {
	return func(w http.ResponseWriter, r *http.Request, handled bool) (bool, error) {
		// Access token was already written to X-Auth-Token header by WriteState
		accessToken := w.Header().Get("X-Auth-Token")

		abUser, err := a.CurrentUser(r)
		if err != nil || abUser == nil {
			return false, err
		}
		user := abUser.(*User)

		// Ensure we have the DB-generated UUID
		if user.ID == "" {
			user.ID, _ = storer.FindIDByPhone(r.Context(), user.Phone)
		}

		// Memberships (empty for new users — they accept an invite separately)
		memberships, _ := storer.LoadMemberships(r.Context(), user.ID)

		// Generate refresh token
		raw, hash, err := GenerateRefreshToken()
		if err != nil {
			return false, err
		}
		deviceType := r.Header.Get("X-Device-Type") // "ios" | "android"
		app := r.Header.Get("X-App")                // "resident" | "guard"
		if err := storer.SaveRefreshToken(r.Context(), user.ID, hash, deviceType, app); err != nil {
			slog.Error("save refresh token", "err", err)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(loginResponse{
			AccessToken:  accessToken,
			RefreshToken: raw,
			ExpiresIn:    900, // 15 min in seconds
			User:         user.ToDTO(),
			Memberships:  memberships,
		}); err != nil {
			return false, err
		}
		return true, nil // mark handled — skip AuthBoss's default redirect
	}
}

// ── Custom handlers ────────────────────────────────────────────────────────────

// handlePhoneOTPSend generates a 6-digit OTP, stores a hashed copy in the
// otps table, and sends it via SMS/WhatsApp (stubbed via SMSMailer).
func handlePhoneOTPSend(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Phone string `json:"phone"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Phone == "" {
			http.Error(w, `{"error":"phone required"}`, http.StatusBadRequest)
			return
		}

		otp, hash, err := generateOTP()
		if err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}

		_, err = db.Exec(r.Context(), `
			INSERT INTO otps (phone, otp_hash, expires_at)
			VALUES ($1, $2, now() + interval '10 minutes')`,
			body.Phone, hash,
		)
		if err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}

		// In dev mode, return OTP for testing; in prod, send via SMS
		if os.Getenv("APP_ENV") == "development" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"otp": otp, "message": "dev mode"})
			return
		}

		slog.Info("OTP stub — would send SMS", "phone", body.Phone, "otp", otp)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"expires_in": 600})
	}
}

// handlePhoneOTPVerify verifies the phone OTP sent during signup to confirm
// the user's phone number, then issues tokens the same way as a normal login.
func handlePhoneOTPVerify(db *pgxpool.Pool, storer *PGStorer, a *ab.Authboss) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Phone string `json:"phone"`
			OTP   string `json:"otp"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}

		// Find latest valid OTP for this phone
		var otpHash string
		var otpID string
		err := db.QueryRow(r.Context(), `
			SELECT id, otp_hash FROM otps
			WHERE phone = $1 AND used_at IS NULL AND expires_at > now()
			ORDER BY created_at DESC LIMIT 1`,
			body.Phone,
		).Scan(&otpID, &otpHash)
		if err != nil {
			http.Error(w, `{"error":"otp not found or expired"}`, http.StatusUnauthorized)
			return
		}

		if !verifyOTPHash(body.OTP, otpHash) {
			http.Error(w, `{"error":"invalid otp"}`, http.StatusUnauthorized)
			return
		}

		// Mark OTP used
		db.Exec(r.Context(), `UPDATE otps SET used_at = now() WHERE id = $1`, otpID)

		// Mark user confirmed
		if err := storer.MarkConfirmed(r.Context(), body.Phone); err != nil {
			http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
			return
		}

		// Load user + memberships + issue tokens (reuse afterAuth logic inline)
		abUser, err := storer.Load(r.Context(), body.Phone)
		if err != nil {
			http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
			return
		}
		user := abUser.(*User)

		// Write auth state so the access token gets generated
		ab.PutSession(w, ab.SessionKey, user.Phone)
		ab.PutSession(w, "uid", user.ID)

		accessToken := w.Header().Get("X-Auth-Token")
		memberships, _ := storer.LoadMemberships(r.Context(), user.ID)

		raw, hash, _ := GenerateRefreshToken()
		storer.SaveRefreshToken(r.Context(), user.ID, hash,
			r.Header.Get("X-Device-Type"), r.Header.Get("X-App"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(loginResponse{
			AccessToken:  accessToken,
			RefreshToken: raw,
			ExpiresIn:    900,
			User:         user.ToDTO(),
			Memberships:  memberships,
		})
	}
}

// handleRefresh exchanges a valid refresh token for a new short-lived access token.
func handleRefresh(_ *pgxpool.Pool, storer *PGStorer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
			http.Error(w, `{"error":"refresh_token required"}`, http.StatusBadRequest)
			return
		}

		userID, err := storer.LoadRefreshToken(r.Context(), body.RefreshToken)
		if err != nil {
			http.Error(w, `{"error":"invalid or expired refresh token"}`, http.StatusUnauthorized)
			return
		}

		accessToken, err := appjwt.Sign(userID)
		if err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": accessToken,
			"expires_in":   900,
		})
	}
}

// handleProfile updates the authenticated user's name and/or email.
func handleProfile(storer *PGStorer, a *ab.Authboss) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		abUser, err := a.CurrentUser(r)
		if err != nil || abUser == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		user := abUser.(*User)

		var body struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		if body.Name != "" {
			user.Name = body.Name
		}
		if body.Email != "" {
			user.Email = &body.Email
		}
		if err := storer.Save(r.Context(), user); err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user.ToDTO())
	}
}

// handleInviteAccept consumes an invite token and creates a membership for
// the authenticated user.
func handleInviteAccept(db *pgxpool.Pool, a *ab.Authboss) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		abUser, err := a.CurrentUser(r)
		if err != nil || abUser == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		user := abUser.(*User)

		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" {
			http.Error(w, `{"error":"token required"}`, http.StatusBadRequest)
			return
		}

		// Load and validate invite
		var inviteID, societyID, flatID, role string
		var invitePhone string
		err = db.QueryRow(r.Context(), `
			SELECT id, society_id, flat_id, role, phone
			FROM invites
			WHERE token = $1 AND accepted_at IS NULL AND expires_at > now()`,
			body.Token,
		).Scan(&inviteID, &societyID, &flatID, &role, &invitePhone)
		if err != nil {
			http.Error(w, `{"error":"invite not found or expired"}`, http.StatusNotFound)
			return
		}

		// Insert membership
		_, err = db.Exec(r.Context(), `
			INSERT INTO memberships (user_id, society_id, flat_id, role, invited_by)
			SELECT $1, $2, $3, $4, invited_by FROM invites WHERE id = $5
			ON CONFLICT (user_id, flat_id, role) DO NOTHING`,
			user.ID, societyID, flatID, role, inviteID,
		)
		if err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}

		// Mark invite accepted
		db.Exec(r.Context(), `UPDATE invites SET accepted_at = now() WHERE id = $1`, inviteID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	}
}
