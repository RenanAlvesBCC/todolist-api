package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/handlers"
	"github.com/RenanAlvesBCC/todolist-api/internal/middleware"
)

func SetupRoutes(router *gin.Engine, authHandler *handlers.AuthHandler) {
	router.GET("/", handlers.HomeHandler)

	router.POST("/register", authHandler.Register)
	router.POST("/login", authHandler.Login)

	// Grupo de rotas protegidas — qualquer rota cadastrada aqui dentro
	// passa primeiro pelo AuthRequired() antes de chegar no handler.
	protected := router.Group("/api")
	protected.Use(middleware.AuthRequired())
	{
		protected.GET("/profile", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"user_id":  c.MustGet("user_id"),
				"username": c.MustGet("username"),
			})
		})
	}
}
