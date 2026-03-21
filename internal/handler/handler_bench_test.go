package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anon-d/urlshortener/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func BenchmarkPostURL(b *testing.B) {
	logger := zap.NewNop().Sugar()
	cache := &mockCacheService{data: make(map[string]string)}
	svc := service.New(cache, nil, logger)
	deleteChan := make(chan DeleteRequest, 1000)
	h := NewURLHandler(svc, "http://localhost:8080", logger, deleteChan, nil)

	b.ResetTimer()
	for b.Loop() {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com/test"))
		h.PostURL(c)
	}
}

func BenchmarkShorten(b *testing.B) {
	logger := zap.NewNop().Sugar()
	cache := &mockCacheService{data: make(map[string]string)}
	svc := service.New(cache, nil, logger)
	deleteChan := make(chan DeleteRequest, 1000)
	h := NewURLHandler(svc, "http://localhost:8080", logger, deleteChan, nil)

	b.ResetTimer()
	for b.Loop() {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(`{"url":"https://example.com/test"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		h.Shorten(c)
	}
}

func BenchmarkGetURL(b *testing.B) {
	logger := zap.NewNop().Sugar()
	cache := &mockCacheService{
		data: map[string]string{
			"abc123": "https://example.com",
		},
	}
	svc := service.New(cache, nil, logger)
	deleteChan := make(chan DeleteRequest, 1000)
	h := NewURLHandler(svc, "http://localhost:8080", logger, deleteChan, nil)

	b.ResetTimer()
	for b.Loop() {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/abc123", nil)
		c.Params = gin.Params{{Key: "id", Value: "abc123"}}
		h.GetURL(c)
	}
}

func BenchmarkBatchShorten(b *testing.B) {
	logger := zap.NewNop().Sugar()
	cache := &mockCacheService{data: make(map[string]string)}
	svc := service.New(cache, &mockStorage{shouldFail: false}, logger)
	deleteChan := make(chan DeleteRequest, 1000)
	h := NewURLHandler(svc, "http://localhost:8080", logger, deleteChan, nil)

	jsonBody := `[{"correlation_id":"1","original_url":"https://example1.com"},{"correlation_id":"2","original_url":"https://example2.com"},{"correlation_id":"3","original_url":"https://example3.com"}]`

	b.ResetTimer()
	for b.Loop() {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")
		h.BatchShorten(c)
	}
}
