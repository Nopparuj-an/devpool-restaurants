package profile

import (
	"restaurants/internal/platform/database"
	"restaurants/internal/profile/repository"
	"restaurants/internal/profile/service"
)

// NewModule builds the profile HTTP handler with its dependencies.
func NewModule(db *database.DB, imageBaseURL string) *Handler {
	return NewHandler(service.New(repository.New(db), imageBaseURL))
}
