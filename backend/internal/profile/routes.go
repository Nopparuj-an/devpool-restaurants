package profile

import (
	"github.com/gin-gonic/gin"

	"restaurants/internal/platform/web"
)

func RegisterRoutes(api *gin.RouterGroup, h *Handler) {
	api.GET("/users/:id", web.Handle(h.Get))
	api.GET("/users/:id/reviews", web.Handle(h.Reviews))
}
