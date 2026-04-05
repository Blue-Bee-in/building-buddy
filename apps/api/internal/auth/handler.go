package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Router() http.Handler {
	r := chi.NewRouter()
	r.Post("/login", handleLogin)
	r.Post("/refresh", handleRefresh)
	r.Post("/register", handleRegister)
	return r
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	// TODO: validate credentials, issue JWT
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "not implemented"})
}

func handleRefresh(w http.ResponseWriter, r *http.Request) {
	// TODO: validate refresh token, issue new access token
	w.WriteHeader(http.StatusNotImplemented)
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	// TODO: validate invite token, create resident user
	w.WriteHeader(http.StatusNotImplemented)
}
