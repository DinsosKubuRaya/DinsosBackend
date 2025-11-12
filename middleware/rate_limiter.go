package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	ginlimiter "github.com/ulule/limiter/v3/drivers/middleware/gin"
	memory "github.com/ulule/limiter/v3/drivers/store/memory"
)

// RateLimiter mengembalikan middleware untuk membatasi request per IP
func RateLimiter() gin.HandlerFunc {
	// 60 request per 1 menit
	rate, err := limiter.NewRateFromFormatted("60-M")
	if err != nil {
		panic(err)
	}

	// Simpan di memory (bisa diganti Redis)
	store := memory.NewStore()

	// Kembalikan middleware limiter
	return ginlimiter.NewMiddleware(limiter.New(store, rate))
}
