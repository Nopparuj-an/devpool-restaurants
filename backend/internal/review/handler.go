// Package review is the ratings and reviews feature (R-REVIEW-*).
//
//	routes.go        URL → handler
//	handler.go       endpoints
//	module.go        wiring
//	model/           Review, Input, errors
//	service/         one-review rule, list filters + Repository port (totals: DB triggers, ADR-0015)
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
	q := model.ListQuery{Sort: c.Query("sort")}
	q.Limit, _ = strconv.Atoi(c.Query("limit"))
	q.Offset, _ = strconv.Atoi(c.Query("offset"))
	if v := c.Query("rating"); v != "" {
		if n, err := strconv.Atoi(v); err != nil || n == 0 {
			q.Rating = -1 // rejected by the service
		} else {
			q.Rating = n
		}
	}
	viewer, admin := auth.Viewer(c)
	page, err := h.svc.List(c.Request.Context(), viewer, admin, id, q)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, page)
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
