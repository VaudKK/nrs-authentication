package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func RateLimiter() gin.HandlerFunc {
	limiter := rate.NewLimiter(1,100)

	return func(c *gin.Context){
		if limiter.Allow(){
			c.Next()
		}else{
			c.JSON(http.StatusTooManyRequests,gin.H{
				"message": "Too many requests",
			})
		}
	}
}