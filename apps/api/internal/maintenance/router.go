package maintenance

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
	"github.com/Blue-Bee-in/building-buddy/internal/storage"
)

// RouterForTest builds the maintenance router with a caller-supplied auth middleware.
// Use in tests to inject a pre-built RequestContext without a real JWT.
func RouterForTest(db *pgxpool.Pool, riverClient *river.Client[pgx.Tx], sc storage.Client, authMW func(http.Handler) http.Handler) http.Handler {
	h := newHandler(db, riverClient, sc)
	r := chi.NewRouter()
	r.Use(authMW)
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Get("/{requestID}", h.get)
	r.Get("/{requestID}/pdf", h.servePDF)
	r.Post("/{requestID}/submit", h.submit)
	r.With(apimw.RequireAdmin).Post("/{requestID}/approve", h.approve)
	r.With(apimw.RequireAdmin).Post("/{requestID}/reject", h.reject)
	return r
}

func Router(db *pgxpool.Pool, riverClient *river.Client[pgx.Tx], sc storage.Client) http.Handler {
	h := newHandler(db, riverClient, sc)
	r := chi.NewRouter()

	// All maintenance routes require authentication.
	r.Use(apimw.Auth(db))

	// Anyone in the society can create and view requests.
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Get("/{requestID}", h.get)
	r.Get("/{requestID}/pdf", h.servePDF)

	// Submit: any authenticated member (handler verifies created_by = caller).
	// Approve / reject: secretary / admin only.
	r.Post("/{requestID}/submit", h.submit)
	r.With(apimw.RequireAdmin).Post("/{requestID}/approve", h.approve)
	r.With(apimw.RequireAdmin).Post("/{requestID}/reject", h.reject)

	return r
}
