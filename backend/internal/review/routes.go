package review

import (
	"github.com/gin-gonic/gin"

	"restaurants/internal/auth"
	"restaurants/internal/platform/web"
)

func RegisterRoutes(api *gin.RouterGroup, h *Handler) {
	api.GET("/restaurants/:id/reviews", web.Handle(h.List))

	mine := api.Group("/restaurants/:id/reviews/me", auth.RequireLogin())
	mine.GET("", web.Handle(h.Mine))
	mine.PUT("", web.Handle(h.Upsert))
	mine.DELETE("", web.Handle(h.Delete))
}
