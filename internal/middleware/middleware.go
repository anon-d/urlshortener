package middleware

import (
	"compress/gzip"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// gzipWriterPool переиспользует gzip.Writer для снижения аллокаций.
var gzipWriterPool = sync.Pool{
	New: func() any {
		return gzip.NewWriter(nil)
	},
}

// GlobalMiddleware возвращает набор общих мидлваров: отлов паник,
// логирование запросов/ответов, gzip-сжатие и распаковка.
func GlobalMiddleware(logger *zap.SugaredLogger) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		PanicMiddleware(logger),
		RequestMiddleware(logger),
		ResponseMiddleware(logger),
		CompressionResponse(),
		DecompressionRequest(),
	}
}

// PanicMiddleware перехватывает паники и возвращает 500 Internal Server Error.
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

// RequestMiddleware логирует входящие запросы: URL, метод, длительность.
func RequestMiddleware(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		logger.Infow("Request", zap.String("url", "http://"+c.Request.Host+c.Request.URL.String()), "method", c.Request.Method, "duration", duration)
	}
}

// ResponseMiddleware логирует исходящие ответы: HTTP-код и размер тела.
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

// Flush сбрасывает буфер gzip-писателя.
func (g *gzipResponseWriter) Flush() {
	if g.shouldCompress && g.Writer != nil {
		g.Writer.Flush()
	}
}

// Header возвращает HTTP-заголовки ответа.
func (g *gzipResponseWriter) Header() http.Header {
	return g.ResponseWriter.Header()
}

// Write записывает данные, сжимая gzip при необходимости.
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

// CompressionResponse сжимает ответы gzip для клиентов, поддерживающих Accept-Encoding: gzip.
func CompressionResponse() gin.HandlerFunc {
	return func(c *gin.Context) {
		acceptEncoding := c.Request.Header.Get("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "gzip") {
			c.Next()
			return
		}

		wo := c.Writer
		wc := gzipWriterPool.Get().(*gzip.Writer)
		wc.Reset(wo)
		gzWriter := &gzipResponseWriter{Writer: wc, ResponseWriter: wo}
		defer func() {
			if gzWriter.Writer != nil {
				gzWriter.Writer.Close()
				gzipWriterPool.Put(wc)
			}
		}()

		c.Writer = gzWriter

		c.Header("Content-Encoding", "gzip")

		c.Next()
	}
}

// DecompressionRequest распаковывает gzip-тело запроса, если Content-Encoding: gzip.
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

// Константы для работы с идентификацией пользователя.
const (
	// UserIDCookieName — имя HTTP-куки для идентификации пользователя.
	UserIDCookieName = "user_id"
	// UserIDContextKey — ключ в контексте Gin для хранения user_id.
	UserIDContextKey = "user_id"
)

// AuthMiddleware проверяет наличие подписанной куки с user_id.
// Если куки нет или подпись недействительна, создается новый user_id и устанавливается подписанная кука.
func AuthMiddleware(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var userID string

		// получаем, проверяем
		cookie, err := c.Cookie(UserIDCookieName)
		if err == nil && cookie != "" {
			if validUserID, valid := validateSignedValue(cookie, secretKey); valid {
				userID = validUserID
			}
		}

		// если нет, то генерим
		if userID == "" {
			userID = generateUserID()
			signedValue := signValue(userID, secretKey)
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
func signValue(value string, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(value))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return value + "." + signature
}

// validateSignedValue проверяет подпись и возвращает оригинальное значение
func validateSignedValue(signedValue string, secretKey string) (string, bool) {
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
