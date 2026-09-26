package review

import (
	"restaurants/internal/platform/database"
	"restaurants/internal/review/repository"
	"restaurants/internal/review/service"
)

// NewService wires the repository into the service (also used by cmd/seed).
func NewService(db *database.DB) service.Service {
	return service.New(repository.New(db), db)
}

// NewModule builds the review HTTP handler with its dependencies.
func NewModule(db *database.DB) *Handler {
	return NewHandler(NewService(db))
}
