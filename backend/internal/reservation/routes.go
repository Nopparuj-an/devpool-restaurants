package reservation

import (
	"github.com/gin-gonic/gin"

	"restaurants/internal/auth"
	"restaurants/internal/platform/web"
)

func RegisterRoutes(api *gin.RouterGroup, h *Handler) {
	api.GET("/restaurants/:id/availability", web.Handle(h.Availability))

	customer := api.Group("", auth.RequireLogin())
	customer.POST("/restaurants/:id/reservations", web.Handle(h.Create))
	customer.GET("/restaurants/:id/reservations", web.Handle(h.ListForOwner)) // owner only, checked in the service
	customer.GET("/me/reservations", web.Handle(h.ListMine))
	customer.GET("/reservations/:id", web.Handle(h.Get))
	customer.PUT("/reservations/:id", web.Handle(h.Update))
	customer.POST("/reservations/:id/cancel", web.Handle(h.Cancel))
}
