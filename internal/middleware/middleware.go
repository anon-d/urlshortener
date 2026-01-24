package middleware

import (
	"compress/gzip"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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

// отлов паники
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

// служебки
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

// для сжатия
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

// распаковка
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

const (
	UserIDCookieName = "user_id"                                  // куки
	UserIDContextKey = "user_id"                                  // ключ контекста
	secretKey        = "my-super-secret-key-change-in-production" // TOP SECRET
)

// AuthMiddleware проверяет наличие подписанной куки с user_id.
// Если куки нет или подпись недействительна, создается новый user_id и устанавливается подписанная кука.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var userID string

		// получаем, проверяем
		cookie, err := c.Cookie(UserIDCookieName)
		if err == nil && cookie != "" {
			if validUserID, valid := validateSignedValue(cookie); valid {
				userID = validUserID
			}
		}

		// если нет, то генерим
		if userID == "" {
			userID = generateUserID()
			signedValue := signValue(userID)
			c.SetCookie(UserIDCookieName, signedValue, 3600*24*365, "/", "", false, true)
		}
		// в контекст
		c.Set(UserIDContextKey, userID)
		c.Next()
	}
}

// generateUserID генерирует уникальный идентификатор пользователя
func generateUserID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// signValue подписывает значение с помощью HMAC-SHA256
func signValue(value string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(value))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return value + "." + signature
}

// validateSignedValue проверяет подпись и возвращает оригинальное значение
func validateSignedValue(signedValue string) (string, bool) {
	parts := strings.Split(signedValue, ".")
	if len(parts) != 2 {
		return "", false
	}

	value := parts[0]
	providedSignature := parts[1]

	// Вычисляем ожидаемую подпись
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(value))
	expectedSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	if hmac.Equal([]byte(expectedSignature), []byte(providedSignature)) {
		return value, true
	}

	return "", false
}
