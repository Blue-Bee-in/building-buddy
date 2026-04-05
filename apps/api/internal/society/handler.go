package society

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
)

func Router() http.Handler {
	r := chi.NewRouter()
	r.Use(apimw.Auth)

	r.With(apimw.RequireRole("super_admin")).Post("/", handleCreate)
	r.With(apimw.RequireRole("super_admin", "society_admin")).Get("/{societyID}", handleGet)
	r.With(apimw.RequireRole("society_admin")).Put("/{societyID}/bank-details", handleUpdateBankDetails)
	r.With(apimw.RequireRole("society_admin")).Post("/{societyID}/charge-heads", handleCreateChargeHead)
	return r
}

func handleCreate(w http.ResponseWriter, r *http.Request)            { w.WriteHeader(http.StatusNotImplemented) }
func handleGet(w http.ResponseWriter, r *http.Request)               { w.WriteHeader(http.StatusNotImplemented) }
func handleUpdateBankDetails(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotImplemented) }
func handleCreateChargeHead(w http.ResponseWriter, r *http.Request)  { w.WriteHeader(http.StatusNotImplemented) }
