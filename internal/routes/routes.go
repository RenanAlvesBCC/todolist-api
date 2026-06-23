package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/handlers"
	"github.com/RenanAlvesBCC/todolist-api/internal/middleware"
	"github.com/RenanAlvesBCC/todolist-api/internal/repository"
)

func SetupRoutes(router *gin.Engine, authHandler *handlers.AuthHandler, listHandler *handlers.TaskListHandler, secRepo *repository.SecurityRepository) {
	// Headers de segurança em todas as rotas
	router.Use(middleware.SecurityHeaders())

	// Rate limiting global — 60 req/min por IP
	router.Use(middleware.RateLimitGlobal())

	router.GET("/", handlers.HomeHandler)

	// Rate limiting mais restritivo nas rotas de autenticação
	auth := router.Group("/")
	auth.Use(middleware.RateLimitAuth())
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	protected := router.Group("/api")
	protected.Use(middleware.AuthRequired())
	protected.Use(middleware.BlacklistCheck(secRepo))
	{
		protected.POST("/logout", authHandler.Logout)
		protected.GET("/lists", listHandler.List)
		protected.POST("/lists", listHandler.Create)
		protected.PUT("/lists/reorder", listHandler.ReorderLists)
		protected.GET("/lists/:id", listHandler.Get)
		protected.PUT("/lists/:id", listHandler.Update)
		protected.DELETE("/lists/:id", listHandler.Delete)
		protected.POST("/lists/:id/items", listHandler.AddItem)
		protected.PUT("/lists/:id/items/reorder", listHandler.ReorderItems)
		protected.PUT("/lists/:id/items/:itemId", listHandler.UpdateItem)
		protected.DELETE("/lists/:id/items/:itemId", listHandler.DeleteItem)
	}
}
