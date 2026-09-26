// Package server wires routes to handlers.
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"restaurants/internal/auth"
	"restaurants/internal/httpx"
	"restaurants/internal/reservation"
	"restaurants/internal/restaurant"
	"restaurants/internal/storage"
)

type Options struct {
	CookieSecure bool
	Images       storage.Store
	ImageBaseURL string // public prefix for image object keys, e.g. "/images"
}

// New returns the API handler. Every route lives under /api, which
// Next.js rewrites to this service (docs/architecture.md).
func New(db *pgxpool.Pool, opts Options) http.Handler {
	mux := http.NewServeMux()

	authHandler := auth.NewHandler(auth.NewService(db), opts.CookieSecure)
	authHandler.Register(mux)
	restaurant.NewHandler(restaurant.NewService(db, opts.Images, opts.ImageBaseURL)).Register(mux)
	reservation.NewHandler(reservation.NewService(db, opts.ImageBaseURL)).Register(mux)

	mux.Handle("GET /api/healthz", httpx.Handler(func(w http.ResponseWriter, r *http.Request) error {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			return httpx.NewError(http.StatusServiceUnavailable, "db_unavailable", "database unreachable")
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return nil
	}))
	mux.Handle("/api/", httpx.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return httpx.ErrNotFound
	}))

	return authHandler.Middleware(mux)
}
