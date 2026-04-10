package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Blue-Bee-in/building-buddy/pkg/jwt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type contextKey string

const RequestCtxKey contextKey = "reqctx"

// RequestContext holds the resolved identity + society context for a request.
// It is injected by the Auth middleware and read by feature handlers.
type RequestContext struct {
	UserID    string
	SocietyID string   // from X-Society-ID header
	FlatID    string   // from X-Flat-ID header
	Roles     []string // membership roles for (user, flat) — may be empty
	IsAdmin   bool     // true if user has admin/secretary/treasurer/committee role in society
	IsGuard   bool     // true if user has guard role in society
}

// Auth validates the Bearer JWT and resolves society/flat context.
// Handlers read the injected *RequestContext via FromCtx().
//
// X-Society-ID and X-Flat-ID are optional — routes that don't require
// a specific society context (e.g. /auth/*, /health) should not use this
// middleware.
func Auth(db *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			claims, err := jwt.Verify(strings.TrimPrefix(header, "Bearer "))
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			rc := &RequestContext{
				UserID:    claims.UserID,
				SocietyID: r.Header.Get("X-Society-ID"),
				FlatID:    r.Header.Get("X-Flat-ID"),
			}

			// Resolve flat-level membership roles when context is provided
			if rc.SocietyID != "" && rc.FlatID != "" {
				rows, err := db.Query(r.Context(), `
					SELECT role FROM memberships
					WHERE user_id = $1 AND flat_id = $2 AND society_id = $3
					  AND is_active = true`,
					rc.UserID, rc.FlatID, rc.SocietyID,
				)
				if err == nil {
					defer rows.Close()
					for rows.Next() {
						var role string
						if rows.Scan(&role) == nil {
							rc.Roles = append(rc.Roles, role)
						}
					}
				}
			}

			// Resolve society-level staff role when society context is provided
			if rc.SocietyID != "" {
				rows, err := db.Query(r.Context(), `
					SELECT role FROM society_staff
					WHERE user_id = $1 AND society_id = $2 AND is_active = true`,
					rc.UserID, rc.SocietyID,
				)
				if err == nil {
					defer rows.Close()
					for rows.Next() {
						var role string
						if rows.Scan(&role) == nil {
							switch role {
							case "admin", "secretary", "treasurer", "committee":
								rc.IsAdmin = true
							case "guard":
								rc.IsGuard = true
							}
						}
					}
				}
			}

			ctx := context.WithValue(r.Context(), RequestCtxKey, rc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// FromCtx returns the RequestContext injected by the Auth middleware.
// Returns nil if called outside an authenticated route.
func FromCtx(r *http.Request) *RequestContext {
	rc, _ := r.Context().Value(RequestCtxKey).(*RequestContext)
	return rc
}

// RequireRoles returns 403 if the authenticated user holds none of the
// specified flat-level roles (e.g. "owner", "resident").
func RequireRoles(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rc := FromCtx(r)
			if rc == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			for _, role := range rc.Roles {
				if _, ok := allowed[role]; ok {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		})
	}
}

// RequireAdmin returns 403 unless the user is a society admin/committee member.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := FromCtx(r)
		if rc == nil || !rc.IsAdmin {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireGuard returns 403 unless the user is a registered guard.
func RequireGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := FromCtx(r)
		if rc == nil || !rc.IsGuard {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
