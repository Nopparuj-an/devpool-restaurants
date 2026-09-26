package admin

import (
	"restaurants/internal/admin/repository"
	"restaurants/internal/admin/service"
	"restaurants/internal/platform/database"
)

// NewModule builds the admin HTTP handler with its dependencies.
func NewModule(db *database.DB) *Handler {
	return NewHandler(service.New(repository.New(db), db))
}
