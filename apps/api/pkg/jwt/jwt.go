// Package jwt issues and verifies short-lived HS256 access tokens.
//
// The JWT carries only the user's UUID (sub claim).  Society/flat context is
// resolved per-request from the X-Society-ID and X-Flat-ID headers by the
// auth middleware — keeping the token thin and supporting users who belong
// to multiple flats across multiple societies without re-login.
package jwt

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const ExpiresIn = 15 * time.Minute

// Claims is the minimal payload embedded in every access token.
type Claims struct {
	UserID string `json:"sub"`
	jwt.RegisteredClaims
}

// Sign creates a signed 15-minute access token for userID.
func Sign(userID string) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ExpiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret())
}

// Verify parses and validates a token string, returning the embedded Claims.
func Verify(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return secret(), nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
	)
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}

func secret() []byte { return []byte(os.Getenv("JWT_SECRET")) }
