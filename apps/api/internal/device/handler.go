package device

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
)

type handler struct {
	db *pgxpool.Pool
}

func newHandler(db *pgxpool.Pool) *handler {
	return &handler{db: db}
}

type registerTokenRequest struct {
	Token      string `json:"token"`
	DeviceType string `json:"device_type"` // "ios" | "android"
	App        string `json:"app"`         // "resident" | "guard"
}

func (h *handler) registerToken(w http.ResponseWriter, r *http.Request) {
	rc := apimw.FromCtx(r)

	var req registerTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.Token == "" {
		jsonError(w, "token is required", http.StatusBadRequest)
		return
	}
	if req.App == "" {
		req.App = "resident"
	}

	_, err := h.db.Exec(r.Context(), `
		INSERT INTO fcm_tokens (user_id, token, device_type, app)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, token) DO UPDATE
		  SET device_type = EXCLUDED.device_type,
		      app = EXCLUDED.app,
		      updated_at = now()`,
		rc.UserID, req.Token, req.DeviceType, req.App,
	)
	if err != nil {
		slog.Error("register push token", "user_id", rc.UserID, "err", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) deleteToken(w http.ResponseWriter, r *http.Request) {
	rc := apimw.FromCtx(r)

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		jsonError(w, "token is required", http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(r.Context(),
		`DELETE FROM fcm_tokens WHERE user_id = $1 AND token = $2`,
		rc.UserID, req.Token,
	)
	if err != nil {
		slog.Error("delete push token", "user_id", rc.UserID, "err", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
