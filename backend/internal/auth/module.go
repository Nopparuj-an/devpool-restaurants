package auth

import (
	"restaurants/internal/auth/repository"
	"restaurants/internal/auth/service"
	"restaurants/internal/platform/config"
	"restaurants/internal/platform/database"
)

// NewService wires the repository into the service (also used by cmd/seed).
func NewService(db *database.DB) service.Service {
	return service.New(repository.New(db), db)
}

// NewModule builds the auth HTTP handler with its dependencies.
func NewModule(db *database.DB, secureCookie bool, google config.Google) *Handler {
	return NewHandler(NewService(db), secureCookie, google)
}
