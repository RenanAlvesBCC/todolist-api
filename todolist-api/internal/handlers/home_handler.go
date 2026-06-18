package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HomeHandler responde com uma mensagem simples — serve pra verificar
// se o servidor está de pé.
func HomeHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Hello, World!",
	})
}
