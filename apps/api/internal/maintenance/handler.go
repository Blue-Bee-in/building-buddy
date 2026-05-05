package maintenance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
	"github.com/Blue-Bee-in/building-buddy/internal/pdf"
	"github.com/Blue-Bee-in/building-buddy/internal/storage"
)

type handler struct {
	db          *pgxpool.Pool
	riverClient *river.Client[pgx.Tx]
	storage     storage.Client // nil → serve from local disk (dev only)
}

func newHandler(db *pgxpool.Pool, riverClient *river.Client[pgx.Tx], sc storage.Client) *handler {
	return &handler{db: db, riverClient: riverClient, storage: sc}
}

// ── Create ────────────────────────────────────────────────────────────────────

type createRequest struct {
	Title          string  `json:"title"`
	Description    *string `json:"description"`
	ContractorName string  `json:"contractor_name"`
	QuotedAmount   float64 `json:"quoted_amount"`
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
	if req.Title == "" || req.ContractorName == "" || req.QuotedAmount <= 0 {
		jsonError(w, "title, contractor_name, and quoted_amount are required", http.StatusBadRequest)
		return
	}
	// E9: input length caps
	if len(req.Title) > 255 || len(req.ContractorName) > 255 {
		jsonError(w, "title and contractor_name must be 255 characters or fewer", http.StatusBadRequest)
		return
	}

	var id string
	err := h.db.QueryRow(r.Context(), `
		INSERT INTO maintenance_requests
		  (society_id, title, description, contractor_name, quoted_amount, created_by)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id`,
		rc.SocietyID, req.Title, req.Description, req.ContractorName, req.QuotedAmount, rc.UserID,
	).Scan(&id)
	if err != nil {
		slog.Error("maintenance create", "err", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// E6: audit log
	h.writeAuditLog(r.Context(), id, rc.SocietyID, rc.UserID, "created", nil)

	mr, err := h.getByID(r.Context(), id, rc.SocietyID)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, mr, http.StatusCreated)
}

// ── List ──────────────────────────────────────────────────────────────────────

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	rc := apimw.FromCtx(r)
	if rc.SocietyID == "" {
		jsonError(w, "X-Society-ID header required", http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT mr.id, mr.society_id, mr.title, mr.description, mr.contractor_name,
		       mr.quoted_amount, mr.approved_amount, mr.status, mr.rejection_reason,
		       mr.created_by, mr.approved_by, mr.created_at, mr.updated_at, mr.approved_at,
		       p.storage_path
		FROM maintenance_requests mr
		LEFT JOIN maintenance_request_pdfs p ON p.request_id = mr.id
		WHERE mr.society_id = $1
		ORDER BY mr.created_at DESC
		LIMIT 100`,
		rc.SocietyID,
	)
	if err != nil {
		slog.Error("maintenance list", "err", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []Request
	for rows.Next() {
		var mr Request
		var storagePath *string
		if err := rows.Scan(
			&mr.ID, &mr.SocietyID, &mr.Title, &mr.Description, &mr.ContractorName,
			&mr.QuotedAmount, &mr.ApprovedAmount, &mr.Status, &mr.RejectionReason,
			&mr.CreatedBy, &mr.ApprovedBy, &mr.CreatedAt, &mr.UpdatedAt, &mr.ApprovedAt,
			&storagePath,
		); err != nil {
			continue
		}
		if storagePath != nil {
			u := pdfURL(mr.ID)
			mr.PDFURL = &u
		}
		results = append(results, mr)
	}
	jsonOK(w, results, http.StatusOK)
}

// ── Get ───────────────────────────────────────────────────────────────────────

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	rc := apimw.FromCtx(r)
	id := chi.URLParam(r, "requestID")
	mr, err := h.getByID(r.Context(), id, rc.SocietyID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	jsonOK(w, mr, http.StatusOK)
}

// ── Submit ────────────────────────────────────────────────────────────────────

func (h *handler) submit(w http.ResponseWriter, r *http.Request) {
	rc := apimw.FromCtx(r)
	id := chi.URLParam(r, "requestID")

	tag, err := h.db.Exec(r.Context(), `
		UPDATE maintenance_requests
		SET status = 'submitted', updated_at = now()
		WHERE id = $1 AND society_id = $2 AND created_by = $3 AND status = 'draft'`,
		id, rc.SocietyID, rc.UserID,
	)
	if err != nil || tag.RowsAffected() == 0 {
		jsonError(w, "request not found or not in draft state", http.StatusBadRequest)
		return
	}

	// E6: audit log
	h.writeAuditLog(r.Context(), id, rc.SocietyID, rc.UserID, "submitted", nil)

	mr, _ := h.getByID(r.Context(), id, rc.SocietyID)
	jsonOK(w, mr, http.StatusOK)
}

// ── Approve ───────────────────────────────────────────────────────────────────

type approveRequest struct {
	ApprovedAmount *float64 `json:"approved_amount"`
}

func (h *handler) approve(w http.ResponseWriter, r *http.Request) {
	rc := apimw.FromCtx(r)
	id := chi.URLParam(r, "requestID")

	var req approveRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	// E4: scope initial SELECT to caller's society
	var quotedAmount float64
	err := h.db.QueryRow(r.Context(),
		`SELECT quoted_amount FROM maintenance_requests WHERE id=$1 AND society_id=$2`,
		id, rc.SocietyID,
	).Scan(&quotedAmount)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	approvedAmount := quotedAmount
	if req.ApprovedAmount != nil && *req.ApprovedAmount > 0 {
		approvedAmount = *req.ApprovedAmount
	}

	tag, err := h.db.Exec(r.Context(), `
		UPDATE maintenance_requests
		SET status = 'approved', approved_by = $1, approved_amount = $2,
		    approved_at = now(), updated_at = now()
		WHERE id = $3 AND society_id = $4 AND status = 'submitted'`,
		rc.UserID, approvedAmount, id, rc.SocietyID,
	)
	if err != nil || tag.RowsAffected() == 0 {
		jsonError(w, "request not found or not in submitted state", http.StatusBadRequest)
		return
	}

	// E6: audit log
	h.writeAuditLog(r.Context(), id, rc.SocietyID, rc.UserID, "approved", nil)

	// Enqueue PDF generation job.
	if _, err := h.riverClient.Insert(r.Context(), PDFJobArgs{RequestID: id}, nil); err != nil {
		slog.Error("enqueue pdf job", "request_id", id, "err", err)
		// Non-fatal — approval still succeeded.
	}

	mr, _ := h.getByID(r.Context(), id, rc.SocietyID)
	jsonOK(w, mr, http.StatusOK)
}

// ── Reject ────────────────────────────────────────────────────────────────────

type rejectRequest struct {
	Reason string `json:"reason"`
}

func (h *handler) reject(w http.ResponseWriter, r *http.Request) {
	rc := apimw.FromCtx(r)
	id := chi.URLParam(r, "requestID")

	var req rejectRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	// E8: require a non-empty rejection reason
	if req.Reason == "" {
		jsonError(w, "reason is required when rejecting a request", http.StatusBadRequest)
		return
	}

	tag, err := h.db.Exec(r.Context(), `
		UPDATE maintenance_requests
		SET status = 'rejected', rejection_reason = $1, updated_at = now()
		WHERE id = $2 AND society_id = $3 AND status = 'submitted'`,
		req.Reason, id, rc.SocietyID,
	)
	if err != nil || tag.RowsAffected() == 0 {
		jsonError(w, "request not found or not in submitted state", http.StatusBadRequest)
		return
	}

	// E6: audit log
	h.writeAuditLog(r.Context(), id, rc.SocietyID, rc.UserID, "rejected", map[string]string{"reason": req.Reason})

	mr, _ := h.getByID(r.Context(), id, rc.SocietyID)
	jsonOK(w, mr, http.StatusOK)
}

// ── PDF serve ─────────────────────────────────────────────────────────────────

func (h *handler) servePDF(w http.ResponseWriter, r *http.Request) {
	rc := apimw.FromCtx(r)
	id := chi.URLParam(r, "requestID")

	// E1: join maintenance_requests to enforce society_id scope
	var storagePath string
	err := h.db.QueryRow(r.Context(), `
		SELECT p.storage_path
		FROM maintenance_request_pdfs p
		JOIN maintenance_requests mr ON mr.id = p.request_id
		WHERE p.request_id = $1 AND mr.society_id = $2`,
		id, rc.SocietyID,
	).Scan(&storagePath)
	if err != nil {
		jsonError(w, "PDF not yet generated", http.StatusNotFound)
		return
	}

	mr, err := h.getByID(r.Context(), id, rc.SocietyID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	filename := fmt.Sprintf("MR-%s-%s.pdf",
		mr.CreatedAt.Format("2006"),
		sanitizeSlug(mr.Title),
	)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	// E5: if storage client is configured, redirect to a presigned URL;
	// otherwise fall back to serving from local disk (dev only).
	if h.storage != nil {
		url, err := h.storage.PresignURL(r.Context(), storagePath, 15*time.Minute)
		if err != nil {
			slog.Error("presign pdf url", "request_id", id, "err", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, url, http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	http.ServeFile(w, r, storagePath)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// E2: getByID now requires societyID — prevents cross-society data exposure.
func (h *handler) getByID(ctx context.Context, id, societyID string) (*Request, error) {
	var mr Request
	var storagePath *string
	err := h.db.QueryRow(ctx, `
		SELECT mr.id, mr.society_id, mr.title, mr.description, mr.contractor_name,
		       mr.quoted_amount, mr.approved_amount, mr.status, mr.rejection_reason,
		       mr.created_by, mr.approved_by, mr.created_at, mr.updated_at, mr.approved_at,
		       p.storage_path
		FROM maintenance_requests mr
		LEFT JOIN maintenance_request_pdfs p ON p.request_id = mr.id
		WHERE mr.id = $1 AND mr.society_id = $2`, id, societyID,
	).Scan(
		&mr.ID, &mr.SocietyID, &mr.Title, &mr.Description, &mr.ContractorName,
		&mr.QuotedAmount, &mr.ApprovedAmount, &mr.Status, &mr.RejectionReason,
		&mr.CreatedBy, &mr.ApprovedBy, &mr.CreatedAt, &mr.UpdatedAt, &mr.ApprovedAt,
		&storagePath,
	)
	if err != nil {
		return nil, err
	}
	if storagePath != nil {
		u := pdfURL(mr.ID)
		mr.PDFURL = &u
	}
	return &mr, nil
}

func pdfURL(requestID string) string {
	return fmt.Sprintf("/api/v1/maintenance-requests/%s/pdf", requestID)
}

// E11: sanitizeSlug returns a URL-safe slug; falls back to "untitled" for
// non-ASCII titles (e.g. Devanagari, Tamil) that produce no ASCII characters.
func sanitizeSlug(s string) string {
	out := make([]byte, 0, len(s))
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			out = append(out, c)
		} else if c >= 'A' && c <= 'Z' {
			out = append(out, c+32)
		} else if c == ' ' || c == '-' {
			out = append(out, '-')
		}
	}
	if len(out) > 30 {
		out = out[:30]
	}
	if len(out) == 0 {
		return "untitled"
	}
	return string(out)
}

// pdfStoragePath returns the local filesystem path for a generated PDF (dev fallback).
func pdfStoragePath(requestID string) string {
	dir := os.Getenv("PDF_STORAGE_DIR")
	if dir == "" {
		dir = "./data/pdfs"
	}
	return filepath.Join(dir, requestID+".pdf")
}

// E10: approved_at is scanned as *time.Time to prevent nil-dereference crash
// when the River job retries on a non-approved request.
func fetchRequestForPDF(ctx context.Context, pool *pgxpool.Pool, id string) (*pdf.MaintenanceRequestData, error) {
	var d pdf.MaintenanceRequestData
	var approvedAt *time.Time
	var approvedAmount float64
	var desc, approvedByName *string

	err := pool.QueryRow(ctx, `
		SELECT
		  mr.id, s.name,
		  mr.title, mr.description,
		  mr.contractor_name,
		  mr.quoted_amount, mr.approved_amount,
		  creator.name,
		  approver.name,
		  mr.approved_at
		FROM maintenance_requests mr
		JOIN societies s ON s.id = mr.society_id
		JOIN users creator ON creator.id = mr.created_by
		LEFT JOIN users approver ON approver.id = mr.approved_by
		WHERE mr.id = $1`, id,
	).Scan(
		&d.ReferenceID, &d.SocietyName,
		&d.Title, &desc,
		&d.ContractorName,
		&d.QuotedAmount, &approvedAmount,
		&d.CreatedByName, &approvedByName,
		&approvedAt,
	)
	if err != nil {
		return nil, err
	}
	if approvedAt == nil {
		return nil, fmt.Errorf("request %s is not yet approved", id)
	}

	d.ReferenceID = fmt.Sprintf("MR-%s-%s", approvedAt.Format("2006"), d.ReferenceID[:8])
	d.ApprovedAmount = approvedAmount
	d.ApprovedAt = *approvedAt
	d.GeneratedAt = time.Now()
	if desc != nil {
		d.Description = *desc
	}
	if approvedByName != nil {
		d.ApprovedByName = *approvedByName
	}
	return &d, nil
}

// writeAuditLog inserts an audit event as best-effort (errors are only logged).
func (h *handler) writeAuditLog(ctx context.Context, requestID, societyID, actorID, action string, meta any) {
	var metaJSON []byte
	if meta != nil {
		metaJSON, _ = json.Marshal(meta)
	}
	if _, err := h.db.Exec(ctx, `
		INSERT INTO maintenance_request_audit_log (request_id, society_id, actor_id, action, metadata)
		VALUES ($1, $2, $3, $4, $5)`,
		requestID, societyID, actorID, action, metaJSON,
	); err != nil {
		slog.Error("audit log insert", "request_id", requestID, "action", action, "err", err)
	}
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
