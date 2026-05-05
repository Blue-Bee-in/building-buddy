package notices

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
)

func Router(db *pgxpool.Pool, riverClient *river.Client[pgx.Tx]) http.Handler {
	h := newHandler(db, riverClient)
	r := chi.NewRouter()

	r.Use(apimw.Auth(db))

	// Secretary only: create broadcast.
	r.With(apimw.RequireAdmin).Post("/", h.create)

	// Any authenticated member: read notices for their society.
	r.Get("/", h.list)
	r.Get("/{noticeID}", h.get)

	return r
}
