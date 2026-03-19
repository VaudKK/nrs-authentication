package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func Logger(log *logrus.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()

		ctx.Next()

		duration := time.Since(start).Milliseconds()

		log.WithFields(logrus.Fields{
			"method":  ctx.Request.Method,
			"path":    ctx.Request.URL.Path,
			"status":  ctx.Writer.Status(),
			"latency": duration,
			"ip":      ctx.ClientIP(),
		}).Info("request completed")
	}
}
