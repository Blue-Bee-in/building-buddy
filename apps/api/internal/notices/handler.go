package notices

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
)

type handler struct {
	db          *pgxpool.Pool
	riverClient *river.Client[pgx.Tx]
}

func newHandler(db *pgxpool.Pool, riverClient *river.Client[pgx.Tx]) *handler {
	return &handler{db: db, riverClient: riverClient}
}

// ── Create (secretary only) ───────────────────────────────────────────────────

type createRequest struct {
	Title      string  `json:"title"`
	Body       string  `json:"body"`
	Category   string  `json:"category"`
	TargetFlat *string `json:"target_flat,omitempty"`
	IsPinned   bool    `json:"is_pinned"`
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	rc := apimw.FromCtx(r)
	if rc.SocietyID == "" {
		jsonError(w, "X-Society-ID header required", http.StatusBadRequest)
		return
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.Title == "" || req.Body == "" {
		jsonError(w, "title and body are required", http.StatusBadRequest)
		return
	}
	// E9: input length caps
	if len(req.Title) > 255 {
		jsonError(w, "title must be 255 characters or fewer", http.StatusBadRequest)
		return
	}
	if len(req.Body) > 10000 {
		jsonError(w, "body must be 10,000 characters or fewer", http.StatusBadRequest)
		return
	}
	if req.Category == "" {
		req.Category = CategoryGeneral
	}

	now := time.Now()
	var id string
	err := h.db.QueryRow(r.Context(), `
		INSERT INTO notices
		  (society_id, created_by, title, body, category, target_flat, is_pinned, published_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id`,
		rc.SocietyID, rc.UserID, req.Title, req.Body, req.Category,
		req.TargetFlat, req.IsPinned, now,
	).Scan(&id)
	if err != nil {
		slog.Error("notice create", "err", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Enqueue push notification job.
	if _, err := h.riverClient.Insert(r.Context(), PushJobArgs{
		NoticeID:  id,
		SocietyID: rc.SocietyID,
		Title:     req.Title,
		Body:      req.Body,
		TargetFlat: req.TargetFlat,
	}, nil); err != nil {
		slog.Error("enqueue push job", "notice_id", id, "err", err)
		// Non-fatal.
	}

	notice, _ := h.getByID(r.Context(), id)
	jsonOK(w, notice, http.StatusCreated)
}

// ── List ──────────────────────────────────────────────────────────────────────

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	rc := apimw.FromCtx(r)
	if rc.SocietyID == "" {
		jsonError(w, "X-Society-ID header required", http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT n.id, n.society_id, n.created_by, n.title, n.body, n.category,
		       n.target_flat, n.is_pinned, n.published_at, n.expires_at, n.created_at,
		       COUNT(notif.id) FILTER (WHERE notif.is_read = true) AS delivered_count,
		       COUNT(notif.id) FILTER (WHERE notif.is_read = false) AS pending_count
		FROM notices n
		LEFT JOIN notifications notif ON notif.ref_id = n.id AND notif.type = 'notice'
		WHERE n.society_id = $1
		  AND (n.expires_at IS NULL OR n.expires_at > now())
		GROUP BY n.id
		ORDER BY n.is_pinned DESC, n.created_at DESC
		LIMIT 100`,
		rc.SocietyID,
	)
	if err != nil {
		slog.Error("notice list", "err", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []Notice
	for rows.Next() {
		var n Notice
		if err := rows.Scan(
			&n.ID, &n.SocietyID, &n.CreatedBy, &n.Title, &n.Body, &n.Category,
			&n.TargetFlat, &n.IsPinned, &n.PublishedAt, &n.ExpiresAt, &n.CreatedAt,
			&n.DeliveredCount, &n.FailedCount,
		); err != nil {
			continue
		}
		results = append(results, n)
	}
	jsonOK(w, results, http.StatusOK)
}

// ── Get ───────────────────────────────────────────────────────────────────────

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "noticeID")
	n, err := h.getByID(r.Context(), id)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	jsonOK(w, n, http.StatusOK)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (h *handler) getByID(ctx context.Context, id string) (*Notice, error) {
	var n Notice
	err := h.db.QueryRow(ctx, `
		SELECT id, society_id, created_by, title, body, category,
		       target_flat, is_pinned, published_at, expires_at, created_at
		FROM notices WHERE id = $1`, id,
	).Scan(
		&n.ID, &n.SocietyID, &n.CreatedBy, &n.Title, &n.Body, &n.Category,
		&n.TargetFlat, &n.IsPinned, &n.PublishedAt, &n.ExpiresAt, &n.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func jsonOK(w http.ResponseWriter, v any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
