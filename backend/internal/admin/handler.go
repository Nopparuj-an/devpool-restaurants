// Package admin is moderation for admins (R-ADMIN-*): list users and
// restaurants, see what a user owns, edit a user's profile, and ban or unban
// either. Admins edit restaurants through the restaurant routes (R-ADMIN-6);
// impersonation is in the auth module, since it swaps the session cookie. Admin rights
// are granted in the database only (`make admin EMAIL=…`).
//
//	routes.go        URL → handler (all behind RequireLogin + RequireAdmin)
//	handler.go       endpoints
//	module.go        wiring
//	model/           User, Restaurant, errors
//	service/         ban rules + Repository port
//	repository/      SQL
package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"restaurants/internal/admin/model"
	"restaurants/internal/admin/service"
	"restaurants/internal/auth"
	"restaurants/internal/platform/web"
)

type Handler struct {
	svc service.Service
}

func NewHandler(svc service.Service) *Handler { return &Handler{svc: svc} }

func listQuery(c *gin.Context) model.ListQuery {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	return model.ListQuery{Q: c.Query("q"), Status: c.Query("status"), Limit: limit, Offset: offset}
}

// banReason reads an optional {"reason": "..."} body.
func banReason(c *gin.Context) (string, error) {
	if c.Request.ContentLength == 0 {
		return "", nil
	}
	var in model.BanInput
	if err := web.Decode(c, &in); err != nil {
		return "", err
	}
	return in.Reason, nil
}

func (h *Handler) ListUsers(c *gin.Context) error {
	list, total, err := h.svc.ListUsers(c.Request.Context(), listQuery(c))
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, gin.H{"users": list, "total": total})
	return nil
}

func (h *Handler) GetUser(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	u, err := h.svc.GetUser(c.Request.Context(), id)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, u)
	return nil
}

func (h *Handler) UpdateUser(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	var in model.UserInput
	if err := web.Decode(c, &in); err != nil {
		return err
	}
	u, err := h.svc.UpdateUser(c.Request.Context(), id, in)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, u)
	return nil
}

func (h *Handler) BanUser(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	reason, err := banReason(c)
	if err != nil {
		return err
	}
	u, err := h.svc.BanUser(c.Request.Context(), auth.MustAccount(c).ID, id, reason)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, u)
	return nil
}

func (h *Handler) UnbanUser(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	u, err := h.svc.UnbanUser(c.Request.Context(), auth.MustAccount(c).ID, id)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, u)
	return nil
}

func (h *Handler) ListRestaurants(c *gin.Context) error {
	list, total, err := h.svc.ListRestaurants(c.Request.Context(), listQuery(c))
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, gin.H{"restaurants": list, "total": total})
	return nil
}

func (h *Handler) BanRestaurant(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	reason, err := banReason(c)
	if err != nil {
		return err
	}
	if err := h.svc.BanRestaurant(c.Request.Context(), id, reason); err != nil {
		return err
	}
	c.Status(http.StatusNoContent)
	return nil
}

func (h *Handler) UnbanRestaurant(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.UnbanRestaurant(c.Request.Context(), id); err != nil {
		return err
	}
	c.Status(http.StatusNoContent)
	return nil
}
