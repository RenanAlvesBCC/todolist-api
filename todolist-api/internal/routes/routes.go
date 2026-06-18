package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/handlers"
)

// SetupRoutes registra todas as rotas da aplicação no router do Gin.
func SetupRoutes(router *gin.Engine) {
	router.GET("/", handlers.HomeHandler)
}
