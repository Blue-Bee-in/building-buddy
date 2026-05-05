package push

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

// PollReceiptsJobArgs triggers an Expo receipt-polling pass.
type PollReceiptsJobArgs struct{}

func (PollReceiptsJobArgs) Kind() string { return "push.poll_receipts" }

// ReceiptPollerWorker polls Expo for delivery receipts and removes dead tokens.
type ReceiptPollerWorker struct {
	river.WorkerDefaults[PollReceiptsJobArgs]
	pool *pgxpool.Pool
}

func NewReceiptPollerWorker(pool *pgxpool.Pool) *ReceiptPollerWorker {
	return &ReceiptPollerWorker{pool: pool}
}

// Work polls tickets created ≥ 25 minutes ago that have not yet been polled.
// Expo guarantees receipts are available within 30 minutes of send.
func (w *ReceiptPollerWorker) Work(ctx context.Context, _ *river.Job[PollReceiptsJobArgs]) error {
	const batchSize = 300 // Expo getReceipts limit

	rows, err := w.pool.Query(ctx, `
		SELECT id, expo_id, token
		FROM push_tickets
		WHERE polled_at IS NULL
		  AND created_at < now() - interval '25 minutes'
		ORDER BY created_at
		LIMIT $1`, batchSize,
	)
	if err != nil {
		return fmt.Errorf("query pending tickets: %w", err)
	}
	defer rows.Close()

	type pendingTicket struct {
		id     string
		expoID string
		token  string
	}
	var pending []pendingTicket
	for rows.Next() {
		var t pendingTicket
		if err := rows.Scan(&t.id, &t.expoID, &t.token); err != nil {
			continue
		}
		pending = append(pending, t)
	}
	rows.Close()

	if len(pending) == 0 {
		return nil
	}

	expoIDs := make([]string, len(pending))
	for i, t := range pending {
		expoIDs[i] = t.expoID
	}

	receipts, err := PollReceipts(ctx, expoIDs)
	if err != nil {
		return fmt.Errorf("expo getReceipts: %w", err)
	}

	now := time.Now()
	var processed []string

	for _, t := range pending {
		receipt, ok := receipts[t.expoID]
		if !ok {
			// Expo hasn't produced a receipt yet; leave for the next pass.
			continue
		}
		processed = append(processed, t.id)

		if receipt.Status == "error" &&
			receipt.Details != nil &&
			receipt.Details.Error == "DeviceNotRegistered" {
			prefix := t.token
			if len(prefix) > 8 {
				prefix = prefix[:8]
			}
			if _, delErr := w.pool.Exec(ctx,
				`DELETE FROM fcm_tokens WHERE token = $1`, t.token,
			); delErr != nil {
				slog.Error("delete dead token", "prefix", prefix, "err", delErr)
			} else {
				slog.Info("deleted DeviceNotRegistered token", "prefix", prefix)
			}
		}
	}

	if len(processed) > 0 {
		if _, err := w.pool.Exec(ctx, `
			UPDATE push_tickets SET polled_at = $1
			WHERE id = ANY($2::uuid[])`,
			now, processed,
		); err != nil {
			return fmt.Errorf("mark polled: %w", err)
		}
	}

	slog.Info("receipt polling done", "checked", len(pending), "processed", len(processed))
	return nil
}

// RunReceiptPoller enqueues a PollReceiptsJobArgs every 5 minutes.
// Run as a goroutine from cmd/worker/main.go alongside the billing scheduler.
func RunReceiptPoller(ctx context.Context, riverClient *river.Client[pgx.Tx]) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := riverClient.Insert(ctx, PollReceiptsJobArgs{}, nil); err != nil {
				slog.Error("enqueue receipt poller", "err", err)
			}
		}
	}
}
