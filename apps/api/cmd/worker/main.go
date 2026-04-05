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

	workers := river.NewWorkers()
	river.AddWorker(workers, billing.NewInvoiceWorker(pool))

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
			"billing":          {MaxWorkers: 5},
		},
		Workers: workers,
	})
	if err != nil {
		slog.Error("failed to create river client", "err", err)
		os.Exit(1)
	}

	// Run the billing scheduler in the background.
	// It enqueues InvoiceJobs daily for all flats due for billing.
	go billing.RunScheduler(ctx, pool, riverClient)

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
