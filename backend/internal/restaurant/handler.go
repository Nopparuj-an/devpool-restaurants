// Package restaurant is the restaurant feature: CRUD, opening hours and
// photos (R-REST-*, R-HOURS-*).
//
//	routes.go            URL → handler
//	handler.go           restaurant endpoints
//	image_handler.go     photo upload, delete, set cover
//	module.go            wiring
//	model/               Input, Summary, Detail, errors
//	service/             rules (Service) + ports (Repository, ImageStore)
//	repository/          SQL, and LoadRules for the booking lock
package restaurant

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"restaurants/internal/auth"
	"restaurants/internal/platform/apperr"
	"restaurants/internal/platform/web"
	"restaurants/internal/restaurant/model"
	"restaurants/internal/restaurant/service"
)

type Handler struct {
	svc service.Service
}

func NewHandler(svc service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) List(c *gin.Context) error {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	list, err := h.svc.List(c.Request.Context(), model.ListQuery{
		Sort: c.Query("sort"), Q: c.Query("q"), Cuisine: c.Query("cuisine"), Limit: limit, Offset: offset,
	})
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, gin.H{"restaurants": list})
	return nil
}

func (h *Handler) Mine(c *gin.Context) error {
	list, err := h.svc.List(c.Request.Context(), model.ListQuery{Sort: "newest", OwnerID: auth.MustAccount(c).ID, Limit: 100})
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, gin.H{"restaurants": list})
	return nil
}

func (h *Handler) Get(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	return h.writeDetail(c, auth.ViewerID(c), id, http.StatusOK)
}

// Create takes multipart/form-data: a "data" field with the Input JSON and
// one or more "images" files (the first becomes the cover).
func (h *Handler) Create(c *gin.Context) error {
	form, err := parseMultipart(c)
	if err != nil {
		return err
	}
	var in model.Input
	dec := json.NewDecoder(strings.NewReader(firstValue(form.Value["data"])))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return apperr.New(apperr.BadRequest, "bad_request", `"data" must be the restaurant JSON: `+err.Error())
	}
	uploads, err := readUploads(form.File["images"])
	if err != nil {
		return err
	}
	me := auth.MustAccount(c).ID
	id, err := h.svc.Create(c.Request.Context(), me, in, uploads)
	if err != nil {
		return err
	}
	return h.writeDetail(c, me, id, http.StatusCreated)
}

func (h *Handler) Update(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	var in model.Input
	if err := web.Decode(c, &in); err != nil {
		return err
	}
	me := auth.MustAccount(c).ID
	if err := h.svc.Update(c.Request.Context(), me, id, in); err != nil {
		return err
	}
	return h.writeDetail(c, me, id, http.StatusOK)
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

func (h *Handler) writeDetail(c *gin.Context, viewer, id int64, status int) error {
	d, err := h.svc.Get(c.Request.Context(), viewer, id)
	if err != nil {
		return err
	}
	c.JSON(status, d)
	return nil
}

func firstValue(v []string) string {
	if len(v) > 0 {
		return v[0]
	}
	return ""
}
