package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Blue-Bee-in/building-buddy/internal/auth"
	"github.com/Blue-Bee-in/building-buddy/internal/billing"
	"github.com/Blue-Bee-in/building-buddy/internal/db"
	"github.com/Blue-Bee-in/building-buddy/internal/device"
	"github.com/Blue-Bee-in/building-buddy/internal/maintenance"
	"github.com/Blue-Bee-in/building-buddy/internal/notices"
	"github.com/Blue-Bee-in/building-buddy/internal/resident"
	"github.com/Blue-Bee-in/building-buddy/internal/society"
	"github.com/Blue-Bee-in/building-buddy/internal/storage"
	"github.com/Blue-Bee-in/building-buddy/internal/visitor"
)

func main() {
	ctx := context.Background()

	pool, err := db.New(ctx)
	if err != nil {
		slog.Error("database init failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// E5: init object storage (DO Spaces). Falls back to nil → local disk in dev.
	sc, err := storage.NewFromEnv()
	if err != nil {
		slog.Warn("object storage unavailable, PDFs will use local disk", "err", err)
		sc = nil
	}

	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Society-ID", "X-Flat-ID", "X-Device-Type", "X-App"},
		ExposedHeaders:   []string{"X-Auth-Token"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Auth routes (AuthBoss + custom)
	authHandler, err := auth.Setup(pool)
	if err != nil {
		slog.Error("auth setup failed", "err", err)
		os.Exit(1)
	}
	r.Mount("/api/v1/auth", authHandler)

	// River client for enqueueing background jobs from HTTP handlers.
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		slog.Error("river client init failed", "err", err)
		os.Exit(1)
	}

	// Feature routes
	r.Mount("/api/v1/societies", society.Router(pool))
	r.Mount("/api/v1/residents", resident.Router(pool))
	r.Mount("/api/v1/billing", billing.Router(pool))
	r.Mount("/api/v1/visitors", visitor.Router(pool))
	r.Mount("/api/v1/maintenance-requests", maintenance.Router(pool, riverClient, sc))
	r.Mount("/api/v1/notices", notices.Router(pool, riverClient))
	r.Mount("/api/v1/devices", device.Router(pool))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("server starting", "port", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
