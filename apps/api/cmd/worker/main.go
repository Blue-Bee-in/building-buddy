package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Blue-Bee-in/building-buddy/internal/billing"
	"github.com/Blue-Bee-in/building-buddy/internal/maintenance"
	"github.com/Blue-Bee-in/building-buddy/internal/notices"
	"github.com/Blue-Bee-in/building-buddy/internal/push"
	"github.com/Blue-Bee-in/building-buddy/internal/storage"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// E5: init object storage (DO Spaces). Falls back to nil → local disk in dev.
	sc, err := storage.NewFromEnv()
	if err != nil {
		slog.Warn("object storage unavailable, PDFs will use local disk", "err", err)
		sc = nil
	}

	workers := river.NewWorkers()
	river.AddWorker(workers, billing.NewInvoiceWorker(pool))
	river.AddWorker(workers, maintenance.NewPDFWorker(pool, sc))
	river.AddWorker(workers, notices.NewPushWorker(pool))
	river.AddWorker(workers, push.NewReceiptPollerWorker(pool)) // E7

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
			"billing":          {MaxWorkers: 5},
			"maintenance":      {MaxWorkers: 3},
			"notices":          {MaxWorkers: 5},
		},
		Workers: workers,
	})
	if err != nil {
		slog.Error("failed to create river client", "err", err)
		os.Exit(1)
	}

	// Background schedulers
	go billing.RunScheduler(ctx, pool, riverClient)
	go push.RunReceiptPoller(ctx, riverClient) // E7: enqueue receipt polling every 5 min

	if err := riverClient.Start(ctx); err != nil {
		slog.Error("failed to start river client", "err", err)
		os.Exit(1)
	}

	slog.Info("worker started")
	<-ctx.Done()
	slog.Info("worker shutting down")

	if err := riverClient.Stop(ctx); err != nil {
		slog.Error("failed to stop river client", "err", err)
	}
}
