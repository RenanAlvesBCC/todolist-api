package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/repository"
)

// BlacklistCheck verifica se o token foi revogado (logout anterior).
// Roda depois do AuthRequired, que já validou a assinatura e expiração.
func BlacklistCheck(secRepo *repository.SecurityRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		parts := strings.Split(header, " ")
		if len(parts) != 2 {
			c.Next()
			return
		}

		token := parts[1]
		blacklisted, err := secRepo.IsTokenBlacklisted(token)
		if err != nil || blacklisted {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token revogado",
			})
			return
		}

		c.Next()
	}
}
