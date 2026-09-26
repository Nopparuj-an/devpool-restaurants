package auth

import (
	"github.com/gin-gonic/gin"

	"restaurants/internal/platform/web"
)

func RegisterRoutes(api *gin.RouterGroup, h *Handler) {
	api.POST("/auth/signup", web.Handle(h.Signup))
	api.POST("/auth/login", web.Handle(h.Login))
	api.POST("/auth/logout", web.Handle(h.Logout))
	api.GET("/auth/providers", web.Handle(h.Providers))
	api.GET("/auth/google/start", web.Handle(h.GoogleStart))
	api.GET("/auth/google/callback", h.GoogleCallback)

	me := api.Group("/me", RequireLogin())
	me.GET("", web.Handle(h.Me))
	me.PUT("/password", web.Handle(h.SetPassword))
}
