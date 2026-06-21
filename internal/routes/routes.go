package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/handlers"
	"github.com/RenanAlvesBCC/todolist-api/internal/middleware"
)

func SetupRoutes(router *gin.Engine, authHandler *handlers.AuthHandler, listHandler *handlers.TaskListHandler) {
	router.GET("/", handlers.HomeHandler)

	router.POST("/register", authHandler.Register)
	router.POST("/login", authHandler.Login)

	protected := router.Group("/api")
	protected.Use(middleware.AuthRequired())
	{
		protected.GET("/lists", listHandler.List)
		protected.POST("/lists", listHandler.Create)
		protected.GET("/lists/:id", listHandler.Get)
		protected.PUT("/lists/:id", listHandler.Update)
		protected.DELETE("/lists/:id", listHandler.Delete)

		protected.POST("/lists/:id/items", listHandler.AddItem)
		protected.PUT("/lists/:id/items/:itemId", listHandler.UpdateItem)
		protected.DELETE("/lists/:id/items/:itemId", listHandler.DeleteItem)
		protected.PUT("/lists/reorder", listHandler.ReorderLists)
		protected.PUT("/lists/:id/items/reorder", listHandler.ReorderItems)
	}
}
