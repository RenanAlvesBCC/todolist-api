package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders adiciona headers HTTP que protegem contra ataques comuns:
// - XSS (Cross-Site Scripting)
// - Clickjacking
// - MIME sniffing
// - Exposição de informações do servidor
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'none'")
		// Remove o header que expõe que o servidor usa Gin/Go
		c.Header("Server", "")
		c.Next()
	}
}
