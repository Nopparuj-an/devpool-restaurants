// Package server wires routes to handlers.
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"restaurants/internal/httpx"
)

type Server struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Server {
	return &Server{db: db}
}

// Routes returns the API handler. Every route lives under /api, which
// Next.js rewrites to this service (docs/architecture.md).
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /api/healthz", httpx.Handler(s.healthz))
	mux.Handle("/api/", httpx.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return httpx.ErrNotFound
	}))
	return mux
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		return httpx.NewError(http.StatusServiceUnavailable, "db_unavailable", "database unreachable")
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	return nil
}
