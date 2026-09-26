package restaurant

import (
	"github.com/gin-gonic/gin"

	"restaurants/internal/auth"
	"restaurants/internal/platform/web"
)

func RegisterRoutes(api *gin.RouterGroup, h *Handler) {
	api.GET("/restaurants", web.Handle(h.List))
	api.GET("/restaurants/:id", web.Handle(h.Get))

	owner := api.Group("", auth.RequireLogin()) // ownership is checked in the service (R-REST-2)
	owner.GET("/me/restaurants", web.Handle(h.Mine))
	owner.POST("/restaurants", web.Handle(h.Create))
	owner.PUT("/restaurants/:id", web.Handle(h.Update))
	owner.DELETE("/restaurants/:id", web.Handle(h.Delete))
	owner.POST("/restaurants/:id/images", web.Handle(h.AddImages))
	owner.DELETE("/restaurants/:id/images/:imageID", web.Handle(h.DeleteImage))
	owner.PUT("/restaurants/:id/images/:imageID/cover", web.Handle(h.SetCover))
}
