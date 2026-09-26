package restaurant

import (
	"restaurants/internal/platform/database"
	"restaurants/internal/restaurant/repository"
	"restaurants/internal/restaurant/service"
)

// NewService wires the repository and image store into the service (also used by cmd/seed).
func NewService(db *database.DB, images service.ImageStore, imageBaseURL string) service.Service {
	return service.New(repository.New(db), images, db, imageBaseURL)
}

// NewModule builds the restaurant HTTP handler with its dependencies.
func NewModule(db *database.DB, images service.ImageStore, imageBaseURL string) *Handler {
	return NewHandler(NewService(db, images, imageBaseURL))
}
