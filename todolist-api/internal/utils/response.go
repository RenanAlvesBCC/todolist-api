package utils

import "github.com/gin-gonic/gin"

// RespondError padroniza o formato de erro devolvido pela API,
// evitando repetir c.JSON(status, gin.H{"error": ...}) em cada handler.
func RespondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
