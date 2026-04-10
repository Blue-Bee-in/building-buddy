package society

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
)

func Router(db *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	r.Use(apimw.Auth(db))

	r.With(apimw.RequireAdmin).Post("/", handleCreate)
	r.With(apimw.RequireAdmin).Get("/{societyID}", handleGet)
	r.With(apimw.RequireAdmin).Put("/{societyID}/bank-details", handleUpdateBankDetails)
	r.With(apimw.RequireAdmin).Post("/{societyID}/charge-heads", handleCreateChargeHead)
	return r
}

func handleCreate(w http.ResponseWriter, r *http.Request)            { w.WriteHeader(http.StatusNotImplemented) }
func handleGet(w http.ResponseWriter, r *http.Request)               { w.WriteHeader(http.StatusNotImplemented) }
func handleUpdateBankDetails(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotImplemented) }
func handleCreateChargeHead(w http.ResponseWriter, r *http.Request)  { w.WriteHeader(http.StatusNotImplemented) }
