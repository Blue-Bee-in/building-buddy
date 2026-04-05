package billing

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

// RunScheduler runs a daily ticker that enqueues InvoiceJobs for all flats
// due for billing. It is intended to run as a goroutine inside the worker binary.
func RunScheduler(ctx context.Context, pool *pgxpool.Pool, client *river.Client[pgx.Tx]) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run once immediately on startup, then on each tick.
	if err := enqueueDueInvoices(ctx, pool, client); err != nil {
		slog.Error("billing scheduler error", "err", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := enqueueDueInvoices(ctx, pool, client); err != nil {
				slog.Error("billing scheduler error", "err", err)
			}
		}
	}
}

func enqueueDueInvoices(ctx context.Context, pool *pgxpool.Pool, client *river.Client[pgx.Tx]) error {
	// TODO: query societies and flats where billing cycle is due today,
	// then batch-insert InvoiceJobArgs into River.
	//
	// Example:
	//   tx, _ := pool.Begin(ctx)
	//   _, err := client.InsertManyTx(ctx, tx, []river.InsertManyParams{{Args: InvoiceJobArgs{...}}})
	//   tx.Commit(ctx)

	_ = pool // used in future implementation
	_ = client
	slog.Info("billing scheduler: checking for due invoices")
	return nil
}
