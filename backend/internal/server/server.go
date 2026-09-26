// Package server is the composition root: it builds the Gin engine and
// plugs every feature module in. To find where a URL is handled, start at a
// feature's routes.go (docs/backend.md has the full map).
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"restaurants/internal/auth"
	"restaurants/internal/platform/apperr"
	"restaurants/internal/platform/config"
	"restaurants/internal/platform/database"
	"restaurants/internal/platform/storage"
	"restaurants/internal/platform/web"
	"restaurants/internal/reservation"
	"restaurants/internal/restaurant"
	"restaurants/internal/review"
)

type Options struct {
	CookieSecure bool
	Google       config.Google // Google login is off when ClientID is empty
	Images       storage.Store
	ImageBaseURL string // public prefix for image object keys, e.g. "/images"
}

// New returns the HTTP handler. Every route lives under /api, which Next.js
// rewrites to this service (docs/architecture.md).
func New(pool *pgxpool.Pool, opts Options) *gin.Engine {
	db := database.New(pool)
	authHandler := auth.NewModule(db, opts.CookieSecure, opts.Google)

	r := gin.New()
	r.Use(gin.Recovery(), requestLog())
	r.NoRoute(func(c *gin.Context) { web.Error(c, apperr.ErrNotFound) })

	api := r.Group("/api", authHandler.Session())
	api.GET("/healthz", web.Handle(func(c *gin.Context) error {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return apperr.New(apperr.Unavailable, "db_unavailable", "database unreachable")
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return nil
	}))

	auth.RegisterRoutes(api, authHandler)
	restaurant.RegisterRoutes(api, restaurant.NewModule(db, opts.Images, opts.ImageBaseURL))
	reservation.RegisterRoutes(api, reservation.NewModule(db, opts.ImageBaseURL))
	review.RegisterRoutes(api, review.NewModule(db))
	return r
}

// requestLog writes one slog line per request.
func requestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("http", "method", c.Request.Method, "path", c.Request.URL.Path,
			"status", c.Writer.Status(), "ms", time.Since(start).Milliseconds())
	}
}
