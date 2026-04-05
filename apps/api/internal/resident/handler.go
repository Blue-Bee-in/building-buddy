package resident

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
)

func Router() http.Handler {
	r := chi.NewRouter()
	r.Use(apimw.Auth)

	r.With(apimw.RequireRole("society_admin")).Post("/invite", handleInvite)
	r.With(apimw.RequireRole("society_admin")).Get("/", handleList)
	r.Get("/{residentID}", handleGet)
	r.Patch("/{residentID}", handleUpdate)

	r.With(apimw.RequireRole("resident")).Post("/domestic-help", handleCreateDomesticHelp)
	r.With(apimw.RequireRole("resident")).Get("/domestic-help", handleListDomesticHelp)
	r.With(apimw.RequireRole("resident")).Post("/vehicles", handleRegisterVehicle)
	return r
}

func handleInvite(w http.ResponseWriter, r *http.Request)           { w.WriteHeader(http.StatusNotImplemented) }
func handleList(w http.ResponseWriter, r *http.Request)             { w.WriteHeader(http.StatusNotImplemented) }
func handleGet(w http.ResponseWriter, r *http.Request)              { w.WriteHeader(http.StatusNotImplemented) }
func handleUpdate(w http.ResponseWriter, r *http.Request)           { w.WriteHeader(http.StatusNotImplemented) }
func handleCreateDomesticHelp(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotImplemented) }
func handleListDomesticHelp(w http.ResponseWriter, r *http.Request)   { w.WriteHeader(http.StatusNotImplemented) }
func handleRegisterVehicle(w http.ResponseWriter, r *http.Request)    { w.WriteHeader(http.StatusNotImplemented) }
