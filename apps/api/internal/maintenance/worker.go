package maintenance

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Blue-Bee-in/building-buddy/internal/pdf"
	"github.com/Blue-Bee-in/building-buddy/internal/storage"
)

// PDFJobArgs holds parameters for a maintenance request PDF generation job.
type PDFJobArgs struct {
	RequestID string `json:"request_id"`
}

func (PDFJobArgs) Kind() string { return "maintenance.pdf" }

// PDFWorker generates the maintenance request PDF and stores the result.
type PDFWorker struct {
	river.WorkerDefaults[PDFJobArgs]
	pool    *pgxpool.Pool
	storage storage.Client // nil → write to local disk (dev only)
}

func NewPDFWorker(pool *pgxpool.Pool, sc storage.Client) *PDFWorker {
	return &PDFWorker{pool: pool, storage: sc}
}

func (w *PDFWorker) Work(ctx context.Context, job *river.Job[PDFJobArgs]) error {
	requestID := job.Args.RequestID
	slog.Info("generating maintenance PDF", "request_id", requestID)

	data, err := fetchRequestForPDF(ctx, w.pool, requestID)
	if err != nil {
		return fmt.Errorf("fetch request: %w", err)
	}

	pdfBytes, err := pdf.MaintenanceRequest(data)
	if err != nil {
		return fmt.Errorf("generate pdf: %w", err)
	}

	// E5: upload to DO Spaces when configured; fall back to local disk in dev.
	var storagePath string
	if w.storage != nil {
		key := fmt.Sprintf("pdfs/%s.pdf", requestID)
		if err := w.storage.Upload(ctx, key, pdfBytes, "application/pdf"); err != nil {
			return fmt.Errorf("upload pdf: %w", err)
		}
		storagePath = key
	} else {
		localPath := pdfStoragePath(requestID)
		if err := os.MkdirAll(storageDirFor(localPath), 0o755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
		if err := os.WriteFile(localPath, pdfBytes, 0o644); err != nil {
			return fmt.Errorf("write pdf: %w", err)
		}
		storagePath = localPath
	}

	_, err = w.pool.Exec(ctx, `
		INSERT INTO maintenance_request_pdfs (request_id, storage_path)
		VALUES ($1, $2)
		ON CONFLICT (request_id) DO UPDATE SET storage_path = $2, generated_at = now()`,
		requestID, storagePath,
	)
	if err != nil {
		return fmt.Errorf("store pdf record: %w", err)
	}

	// E6: audit log — system event, actor_id NULL
	if _, err := w.pool.Exec(ctx, `
		INSERT INTO maintenance_request_audit_log (request_id, society_id, action)
		SELECT $1, society_id, 'pdf_generated' FROM maintenance_requests WHERE id = $1`,
		requestID,
	); err != nil {
		slog.Error("audit log pdf_generated", "request_id", requestID, "err", err)
	}

	slog.Info("maintenance PDF ready", "request_id", requestID, "path", storagePath)
	return nil
}

func storageDirFor(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}
