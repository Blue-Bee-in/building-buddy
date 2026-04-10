package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/Blue-Bee-in/building-buddy/internal/auth"
	"github.com/Blue-Bee-in/building-buddy/internal/db"
	apimw "github.com/Blue-Bee-in/building-buddy/internal/middleware"
)

func main() {
	ctx := context.Background()

	pool, err := db.New(ctx)
	if err != nil {
		slog.Error("database init failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Society-ID", "X-Flat-ID", "X-Device-Type", "X-App"},
		ExposedHeaders:   []string{"X-Auth-Token"}, // AuthBoss writes the session JWT here
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

	// Feature routes — all require a valid JWT via the Auth middleware
	r.Group(func(r chi.Router) {
		r.Use(apimw.Auth(pool))

		// TODO: mount feature routers
		// r.Mount("/api/v1/societies", society.NewHandler(pool).Router())
		// r.Mount("/api/v1/residents", resident.NewHandler(pool).Router())
		// r.Mount("/api/v1/billing",   billing.NewHandler(pool).Router())
		// r.Mount("/api/v1/visitors",  visitor.NewHandler(pool).Router())
	})

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

