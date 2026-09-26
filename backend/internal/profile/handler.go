// Package profile is the public user profile (R-PROFILE-*): a display name,
// when they joined, and their reviews. Their restaurants come from
// GET /api/restaurants?owner_id=.
//
//	routes.go        URL → handler
//	handler.go       endpoints
//	module.go        wiring
//	model/           Profile, Review
//	service/         visibility rules + Repository port
//	repository/      SQL
package profile

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"restaurants/internal/auth"
	"restaurants/internal/platform/web"
	"restaurants/internal/profile/service"
)

type Handler struct {
	svc service.Service
}

func NewHandler(svc service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Get(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	viewer, admin := auth.Viewer(c)
	p, err := h.svc.Get(c.Request.Context(), viewer, admin, id)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, p)
	return nil
}

func (h *Handler) Reviews(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	viewer, admin := auth.Viewer(c)
	list, total, err := h.svc.Reviews(c.Request.Context(), viewer, admin, id, limit, offset)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, gin.H{"reviews": list, "total": total})
	return nil
}
