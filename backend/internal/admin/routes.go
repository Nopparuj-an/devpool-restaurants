package admin

import (
	"github.com/gin-gonic/gin"

	"restaurants/internal/auth"
	"restaurants/internal/platform/web"
)

func RegisterRoutes(api *gin.RouterGroup, h *Handler) {
	g := api.Group("/admin", auth.RequireLogin(), auth.RequireAdmin())
	g.GET("/users", web.Handle(h.ListUsers))
	g.GET("/users/:id", web.Handle(h.GetUser))
	g.PUT("/users/:id", web.Handle(h.UpdateUser))
	g.POST("/users/:id/ban", web.Handle(h.BanUser))
	g.POST("/users/:id/unban", web.Handle(h.UnbanUser))
	g.POST("/users/delete", web.Handle(h.DeleteUsers))
	g.GET("/restaurants", web.Handle(h.ListRestaurants))
	g.POST("/restaurants/:id/ban", web.Handle(h.BanRestaurant))
	g.POST("/restaurants/:id/unban", web.Handle(h.UnbanRestaurant))
	g.POST("/restaurants/delete", web.Handle(h.DeleteRestaurants))
}
