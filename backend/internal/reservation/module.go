package reservation

import (
	"restaurants/internal/platform/database"
	"restaurants/internal/reservation/repository"
	"restaurants/internal/reservation/service"
)

// NewService wires the repository into the service (also used by cmd/seed).
func NewService(db *database.DB, imageBaseURL string) service.Service {
	return service.New(repository.New(db), db, imageBaseURL)
}

// NewModule builds the reservation HTTP handler with its dependencies.
func NewModule(db *database.DB, imageBaseURL string) *Handler {
	return NewHandler(NewService(db, imageBaseURL))
}
