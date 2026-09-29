package admin

import (
	"restaurants/internal/admin/repository"
	"restaurants/internal/admin/service"
	"restaurants/internal/platform/database"
	"restaurants/internal/platform/storage"
)

// NewModule builds the admin HTTP handler with its dependencies.
func NewModule(db *database.DB, images storage.Store) *Handler {
	return NewHandler(service.New(repository.New(db), db, images))
}
