package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func GlobalMiddleware(logger *zap.SugaredLogger) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		PanicMiddleware(logger),
		RequestMiddleware(logger),
		ResponseMiddleware(logger),
		CompressionResponse(),
		DecompressionRequest(),
	}
}

func PanicMiddleware(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorw("Panic recovered", "panic", r)
				c.JSON(http.StatusInternalServerError, gin.H{
					"status":  "Error",
					"message": "Internal server error",
				})
			}
		}()
		c.Next()
	}
}

func RequestMiddleware(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		logger.Infow("Request", zap.String("url", "http://"+c.Request.Host+c.Request.URL.String()), "method", c.Request.Method, "duration", duration)
	}
}

func ResponseMiddleware(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		logger.Infow("Response", "code", c.Writer.Status(), "size", c.Writer.Size())
	}
}

type gzipResponseWriter struct {
	*gzip.Writer
	gin.ResponseWriter
	writeStarted   bool
	shouldCompress bool
}

func (g *gzipResponseWriter) Flush() {
	if g.shouldCompress && g.Writer != nil {
		g.Writer.Flush()
	}
}

func (g *gzipResponseWriter) Header() http.Header {
	return g.ResponseWriter.Header()
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	if !g.writeStarted {
		g.writeStarted = true
		contentType := g.ResponseWriter.Header().Get("Content-Type")

		g.shouldCompress = strings.Contains(contentType, "application/json") ||
			strings.Contains(contentType, "text/html")

		if !g.shouldCompress {
			g.ResponseWriter.Header().Del("Content-Encoding")
			g.Writer = nil
		}
	}

	if g.shouldCompress && g.Writer != nil {
		return g.Writer.Write(b)
	}
	return g.ResponseWriter.Write(b)
}

func CompressionResponse() gin.HandlerFunc {
	return func(c *gin.Context) {
		acceptEncoding := c.Request.Header.Get("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "gzip") {
			c.Next()
			return
		}

		wo := c.Writer
		wc := gzip.NewWriter(wo)
		gzWriter := &gzipResponseWriter{Writer: wc, ResponseWriter: wo}
		defer func() {
			if gzWriter.Writer != nil {
				gzWriter.Writer.Close()
			}
		}()

		c.Writer = gzWriter

		c.Header("Content-Encoding", "gzip")

		c.Next()
	}
}

func DecompressionRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		encodingType := c.Request.Header.Get("Content-Encoding")
		if encodingType == "" {
			c.Next()
			return
		}

		if encodingType == "gzip" {
			body, err := gzip.NewReader(c.Request.Body)
			if err != nil {
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
			defer body.Close()
			c.Request.Body = body
		}
		c.Next()
	}
}
