package billing

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

// InvoiceJobArgs holds the parameters for a single invoice generation job.
type InvoiceJobArgs struct {
	SocietyID string `json:"society_id"`
	FlatID    string `json:"flat_id"`
	Period    string `json:"period"` // "YYYY-MM"
}

func (InvoiceJobArgs) Kind() string { return "invoice.generate" }

// InvoiceWorker processes invoice generation jobs enqueued by the scheduler.
type InvoiceWorker struct {
	river.WorkerDefaults[InvoiceJobArgs]
	pool *pgxpool.Pool
}

func NewInvoiceWorker(pool *pgxpool.Pool) *InvoiceWorker {
	return &InvoiceWorker{pool: pool}
}

func (w *InvoiceWorker) Work(ctx context.Context, job *river.Job[InvoiceJobArgs]) error {
	args := job.Args
	slog.Info("generating invoice", "society_id", args.SocietyID, "flat_id", args.FlatID, "period", args.Period)

	// TODO:
	// 1. Fetch applicable charge heads for the flat
	// 2. Check for overdue invoices and apply penalty
	// 3. Create invoice record in Postgres
	// 4. Generate PDF via gofpdf
	// 5. Upload PDF to DigitalOcean Spaces
	// 6. Send WhatsApp message with invoice link + UPI intent deep-link

	return nil
}
