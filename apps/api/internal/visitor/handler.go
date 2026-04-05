package visitor

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
)

func Router() http.Handler {
	r := chi.NewRouter()
	r.Use(apimw.Auth)

	// Resident: create/list passes
	r.With(apimw.RequireRole("resident")).Post("/passes", handleCreatePass)
	r.With(apimw.RequireRole("resident")).Get("/passes", handleListPasses)
	r.With(apimw.RequireRole("resident")).Delete("/passes/{passID}", handleRevokePass)

	// Delivery passes
	r.With(apimw.RequireRole("resident")).Post("/deliveries", handleCreateDelivery)

	// Guard: validate & log entry/exit
	r.With(apimw.RequireRole("security_guard")).Post("/validate", handleValidatePass)
	r.With(apimw.RequireRole("security_guard")).Post("/entries", handleLogEntry)
	r.With(apimw.RequireRole("security_guard")).Patch("/entries/{entryID}/exit", handleLogExit)
	r.With(apimw.RequireRole("security_guard")).Get("/entries", handleListEntries)

	return r
}

func handleCreatePass(w http.ResponseWriter, r *http.Request)    { w.WriteHeader(http.StatusNotImplemented) }
func handleListPasses(w http.ResponseWriter, r *http.Request)    { w.WriteHeader(http.StatusNotImplemented) }
func handleRevokePass(w http.ResponseWriter, r *http.Request)    { w.WriteHeader(http.StatusNotImplemented) }
func handleCreateDelivery(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotImplemented) }
func handleValidatePass(w http.ResponseWriter, r *http.Request)  { w.WriteHeader(http.StatusNotImplemented) }
func handleLogEntry(w http.ResponseWriter, r *http.Request)      { w.WriteHeader(http.StatusNotImplemented) }
func handleLogExit(w http.ResponseWriter, r *http.Request)       { w.WriteHeader(http.StatusNotImplemented) }
func handleListEntries(w http.ResponseWriter, r *http.Request)   { w.WriteHeader(http.StatusNotImplemented) }
