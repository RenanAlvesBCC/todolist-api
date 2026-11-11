package middleware

import (
	"github.com/gin-gonic/gin"
	limiter "github.com/ulule/limiter/v3"
	ginlimiter "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// RateLimitGlobal conta por IP público: na oficina o painel e os celulares dos
// mecânicos saem pelo mesmo IP, e abrir um veículo dispara ~6 requisições.
func RateLimitGlobal() gin.HandlerFunc {
	rate, _ := limiter.NewRateFromFormatted("600-M")
	store := memory.NewStore()
	instance := limiter.New(store, rate)
	return ginlimiter.NewMiddleware(instance)
}

// RateLimitIntegration: lotes do robô de sincronização (por IP), além do limite global.
func RateLimitIntegration() gin.HandlerFunc {
	rate, _ := limiter.NewRateFromFormatted("30-M")
	store := memory.NewStore()
	instance := limiter.New(store, rate)
	return ginlimiter.NewMiddleware(instance)
}

func RateLimitAuth() gin.HandlerFunc {
	rate, _ := limiter.NewRateFromFormatted("10-M")
	store := memory.NewStore()
	instance := limiter.New(store, rate)
	return ginlimiter.NewMiddleware(instance)
}
