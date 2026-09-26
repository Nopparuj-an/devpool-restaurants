// Package reservation is the booking feature: book, change and cancel a
// table, seats left per slot, and the owner's reservation table
// (R-BOOK-*, R-EDIT-*, R-CANCEL-*, R-SEATS-1).
//
//	routes.go        URL → handler
//	handler.go       endpoints
//	module.go        wiring
//	model/           Input, Reservation, Availability, errors
//	service/         the locked booking flow (ADR-0003) + Repository port
//	repository/      SQL
//
// The seat rules themselves are pure functions in internal/booking.
package reservation

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"restaurants/internal/auth"
	"restaurants/internal/platform/apperr"
	"restaurants/internal/platform/web"
	"restaurants/internal/reservation/model"
	"restaurants/internal/reservation/service"
)

type Handler struct {
	svc service.Service
}

func NewHandler(svc service.Service) *Handler { return &Handler{svc: svc} }

func decodeInput(c *gin.Context) (model.Input, error) {
	var in model.Input
	if err := web.Decode(c, &in); err != nil {
		return in, err
	}
	if in.StartsAt.IsZero() || in.EndsAt.IsZero() {
		return in, model.ErrMissingTimes
	}
	return in, nil
}

func (h *Handler) Create(c *gin.Context) error {
	restaurantID, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	in, err := decodeInput(c)
	if err != nil {
		return err
	}
	me := auth.MustAccount(c).ID
	id, err := h.svc.Create(c.Request.Context(), me, restaurantID, in)
	if err != nil {
		return err
	}
	return h.write(c, me, id, http.StatusCreated)
}

func (h *Handler) Get(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	return h.write(c, auth.MustAccount(c).ID, id, http.StatusOK)
}

func (h *Handler) Update(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	in, err := decodeInput(c)
	if err != nil {
		return err
	}
	me := auth.MustAccount(c).ID
	if err := h.svc.Update(c.Request.Context(), me, id, in); err != nil {
		return err
	}
	return h.write(c, me, id, http.StatusOK)
}

func (h *Handler) Cancel(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	me := auth.MustAccount(c).ID
	if err := h.svc.Cancel(c.Request.Context(), me, id); err != nil {
		return err
	}
	return h.write(c, me, id, http.StatusOK)
}

func (h *Handler) write(c *gin.Context, me, id int64, status int) error {
	res, err := h.svc.Get(c.Request.Context(), me, id)
	if err != nil {
		return err
	}
	c.JSON(status, res)
	return nil
}

func (h *Handler) ListMine(c *gin.Context) error {
	list, err := h.svc.ListMine(c.Request.Context(), auth.MustAccount(c).ID)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, gin.H{"reservations": list})
	return nil
}

// ListForOwner defaults to the next 7 days when from/to are omitted.
func (h *Handler) ListForOwner(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	from, to, err := timeRange(c, 7*24*time.Hour)
	if err != nil {
		return err
	}
	list, err := h.svc.ListForOwner(c.Request.Context(), auth.MustAccount(c).ID, id, from, to)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, gin.H{"reservations": list})
	return nil
}

// Availability defaults to the next 24 hours when from/to are omitted.
func (h *Handler) Availability(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	from, to, err := timeRange(c, 24*time.Hour)
	if err != nil {
		return err
	}
	a, err := h.svc.Availability(c.Request.Context(), id, from, to)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, a)
	return nil
}

// timeRange reads ?from=&to= (RFC 3339); from defaults to now, to to from+span.
func timeRange(c *gin.Context, span time.Duration) (time.Time, time.Time, error) {
	parse := func(key string, fallback time.Time) (time.Time, error) {
		v := c.Query(key)
		if v == "" {
			return fallback, nil
		}
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return t, apperr.InvalidInput("%s must be an RFC 3339 timestamp", key)
		}
		return t, nil
	}
	from, err := parse("from", time.Now())
	if err != nil {
		return from, from, err
	}
	to, err := parse("to", from.Add(span))
	return from, to, err
}
