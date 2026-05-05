package device

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
)

func Router(db *pgxpool.Pool) http.Handler {
	h := newHandler(db)
	r := chi.NewRouter()

	r.Use(apimw.Auth(db))

	r.Post("/push-token", h.registerToken)
	r.Delete("/push-token", h.deleteToken)

	return r
}
