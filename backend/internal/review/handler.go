// Package review is the ratings and reviews feature (R-REVIEW-*).
//
//	routes.go        URL → handler
//	handler.go       endpoints
//	module.go        wiring
//	model/           Review, Input, errors
//	service/         one-review rule, totals in the same transaction + Repository port
//	repository/      SQL
package review

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"restaurants/internal/auth"
	"restaurants/internal/platform/web"
	"restaurants/internal/review/model"
	"restaurants/internal/review/service"
)

type Handler struct {
	svc service.Service
}

func NewHandler(svc service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) List(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	list, err := h.svc.List(c.Request.Context(), auth.ViewerID(c), id, limit, offset)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, gin.H{"reviews": list})
	return nil
}

func (h *Handler) Mine(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	rv, err := h.svc.Mine(c.Request.Context(), auth.MustAccount(c).ID, id)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, rv)
	return nil
}

// Upsert answers 201 when the review is new and 200 when it replaced mine.
func (h *Handler) Upsert(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	var in model.Input
	if err := web.Decode(c, &in); err != nil {
		return err
	}
	me := auth.MustAccount(c).ID
	created, err := h.svc.Upsert(c.Request.Context(), me, id, in)
	if err != nil {
		return err
	}
	rv, err := h.svc.Mine(c.Request.Context(), me, id)
	if err != nil {
		return err
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, rv)
	return nil
}

func (h *Handler) Delete(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.Request.Context(), auth.MustAccount(c).ID, id); err != nil {
		return err
	}
	c.Status(http.StatusNoContent)
	return nil
}
