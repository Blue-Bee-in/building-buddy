package resident

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
)

func Router(db *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	r.Use(apimw.Auth(db))

	r.With(apimw.RequireAdmin).Post("/invite", handleInvite)
	r.With(apimw.RequireAdmin).Get("/", handleList)
	r.Get("/{residentID}", handleGet)
	r.Patch("/{residentID}", handleUpdate)

	r.With(apimw.RequireRoles("resident", "owner")).Post("/domestic-help", handleCreateDomesticHelp)
	r.With(apimw.RequireRoles("resident", "owner")).Get("/domestic-help", handleListDomesticHelp)
	r.With(apimw.RequireRoles("resident", "owner")).Post("/vehicles", handleRegisterVehicle)
	return r
}

func handleInvite(w http.ResponseWriter, r *http.Request)           { w.WriteHeader(http.StatusNotImplemented) }
func handleList(w http.ResponseWriter, r *http.Request)             { w.WriteHeader(http.StatusNotImplemented) }
func handleGet(w http.ResponseWriter, r *http.Request)              { w.WriteHeader(http.StatusNotImplemented) }
func handleUpdate(w http.ResponseWriter, r *http.Request)           { w.WriteHeader(http.StatusNotImplemented) }
func handleCreateDomesticHelp(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotImplemented) }
func handleListDomesticHelp(w http.ResponseWriter, r *http.Request)   { w.WriteHeader(http.StatusNotImplemented) }
func handleRegisterVehicle(w http.ResponseWriter, r *http.Request)    { w.WriteHeader(http.StatusNotImplemented) }
