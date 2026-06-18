package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/handlers"
	"github.com/RenanAlvesBCC/todolist-api/internal/middleware"
)

func SetupRoutes(router *gin.Engine, authHandler *handlers.AuthHandler, taskHandler *handlers.TaskHandler) {
	router.GET("/", handlers.HomeHandler)

	router.POST("/register", authHandler.Register)
	router.POST("/login", authHandler.Login)

	protected := router.Group("/api")
	protected.Use(middleware.AuthRequired())
	{
		protected.POST("/tasks", taskHandler.Create)
		protected.GET("/tasks", taskHandler.List)
		protected.GET("/tasks/:id", taskHandler.Get)
		protected.PUT("/tasks/:id", taskHandler.Update)
		protected.DELETE("/tasks/:id", taskHandler.Delete)
	}
}
