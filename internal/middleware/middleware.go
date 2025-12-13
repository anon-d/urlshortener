package middleware

import (
	"time"

	"github.com/anon-d/urlshortener/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RequestMiddleware(logger *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		logger.ZLog.Infow("Request", zap.String("url", "http://"+c.Request.Host+c.Request.URL.String()), "method", c.Request.Method, "duration", duration)
	}
}

func ResponseMiddleware(logger *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		logger.ZLog.Infow("Response", "code", c.Writer.Status(), "size", c.Writer.Size())
	}
}
