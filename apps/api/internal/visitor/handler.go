package visitor

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
)

func Router(db *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	r.Use(apimw.Auth(db))

	// Resident: create/list passes
	r.With(apimw.RequireRoles("resident", "owner")).Post("/passes", handleCreatePass)
	r.With(apimw.RequireRoles("resident", "owner")).Get("/passes", handleListPasses)
	r.With(apimw.RequireRoles("resident", "owner")).Delete("/passes/{passID}", handleRevokePass)

	// Delivery passes
	r.With(apimw.RequireRoles("resident", "owner")).Post("/deliveries", handleCreateDelivery)

	// Guard: validate & log entry/exit
	r.With(apimw.RequireGuard).Post("/validate", handleValidatePass)
	r.With(apimw.RequireGuard).Post("/entries", handleLogEntry)
	r.With(apimw.RequireGuard).Patch("/entries/{entryID}/exit", handleLogExit)
	r.With(apimw.RequireGuard).Get("/entries", handleListEntries)

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
