package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anon-d/urlshortener/internal/model"
	"github.com/anon-d/urlshortener/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Mock implementations
type mockCacheService struct {
	data map[string]any
}

func (m *mockCacheService) Set(data *model.Data) {
	if m.data == nil {
		m.data = make(map[string]any)
	}
	m.data[data.ID] = data.OriginalURL
}

func (m *mockCacheService) Get(id string) (any, bool) {
	if m.data == nil {
		return nil, false
	}
	val, ok := m.data[id]
	return val, ok
}

func (m *mockCacheService) Self() []model.Data {
	return []model.Data{}
}

type mockStorage struct {
	shouldFail bool
}

func (m *mockStorage) Insert(ctx context.Context, data model.Data) error {
	if m.shouldFail {
		return errors.New("storage error")
	}
	return nil
}

func (m *mockStorage) InsertBatch(ctx context.Context, dataList []model.Data) error {
	if m.shouldFail {
		return errors.New("storage batch error")
	}
	return nil
}

func (m *mockStorage) Select(ctx context.Context) ([]model.Data, error) {
	return []model.Data{}, nil
}

func (m *mockStorage) GetURLByOriginal(ctx context.Context, originalURL string) (string, error) {
	if m.shouldFail {
		return "", errors.New("get url error")
	}
	return "existing-short-url", nil
}

func (m *mockStorage) GetURLsByUser(ctx context.Context, userID string) ([]model.Data, error) {
	if m.shouldFail {
		return nil, errors.New("get urls by user error")
	}
	return []model.Data{
		{
			ID:          "abc123",
			ShortURL:    "abc123",
			OriginalURL: "https://example1.com",
			UserID:      userID,
		},
		{
			ID:          "def456",
			ShortURL:    "def456",
			OriginalURL: "https://example2.com",
			UserID:      userID,
		},
	}, nil
}

func (m *mockStorage) Ping(ctx context.Context) error {
	if m.shouldFail {
		return errors.New("ping error")
	}
	return nil
}

func TestPostURL_Success(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, nil, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := strings.NewReader("https://example.com")
	c.Request = httptest.NewRequest(http.MethodPost, "/", body)

	handler.PostURL(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	respBody := w.Body.String()
	if respBody == "" {
		t.Errorf("expected response to contain short URL, got empty string")
	}
}

func TestPostURL_EmptyBody(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, nil, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	handler.PostURL(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPostURL_DiskError(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, &mockStorage{shouldFail: true}, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := strings.NewReader("https://example.com")
	c.Request = httptest.NewRequest(http.MethodPost, "/", body)

	handler.PostURL(c)

	// Disk fails, but URL is still in cache, so request succeeds
	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestPostURL_WithDB_Success(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, &mockStorage{shouldFail: false}, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := strings.NewReader("https://example.com")
	c.Request = httptest.NewRequest(http.MethodPost, "/", body)

	handler.PostURL(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestPostURL_WithDB_Error(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, &mockStorage{shouldFail: true}, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := strings.NewReader("https://example.com")
	c.Request = httptest.NewRequest(http.MethodPost, "/", body)

	handler.PostURL(c)

	// DB fails, but falls back to disk storage successfully
	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestGetURL_Success(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{
		data: map[string]any{
			"abc123": "https://example.com",
		},
	}

	svc := service.New(cache, nil, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/abc123", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc123"}}

	handler.GetURL(c)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected status %d, got %d", http.StatusTemporaryRedirect, w.Code)
	}

	location := w.Header().Get("Location")
	if location != "https://example.com" {
		t.Errorf("expected Location header 'https://example.com', got %s", location)
	}
}

func TestGetURL_NotFound(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, nil, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.GetURL(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetURL_EmptyID(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, nil, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.GetURL(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestShorten_Success(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, nil, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonBody := `{"url":"https://example.com"}`
	body := strings.NewReader(jsonBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/shorten", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Shorten(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %s", contentType)
	}

	respBody := w.Body.String()
	if !strings.Contains(respBody, "result") {
		t.Errorf("expected response to contain 'result' field, got %s", respBody)
	}
}

func TestShorten_InvalidJSON(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, nil, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := strings.NewReader(`{"url":}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/shorten", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Shorten(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestShorten_MissingURL(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, nil, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonBody := `{}`
	body := strings.NewReader(jsonBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/shorten", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Shorten(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPingDB_Success(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, &mockStorage{shouldFail: false}, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodGet, "/ping", nil)

	handler.PingDB(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestPingDB_Error(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, &mockStorage{shouldFail: false}, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodGet, "/ping", nil)

	handler.PingDB(c)

	// PingDB returns 200 even on error because fallback storage exists
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestPingDB_DBNotConnected(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, nil, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodGet, "/ping", nil)

	handler.PingDB(c)

	// PingDB returns 500 when storage is not initialized
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestBatchShorten_Success(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, &mockStorage{shouldFail: false}, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonBody := `[{"correlation_id":"1","original_url":"https://example1.com"},{"correlation_id":"2","original_url":"https://example2.com"}]`
	body := strings.NewReader(jsonBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/shorten/batch", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchShorten(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %s", contentType)
	}

	respBody := w.Body.String()
	if !strings.Contains(respBody, "correlation_id") {
		t.Errorf("expected response to contain 'correlation_id' field, got %s", respBody)
	}
	if !strings.Contains(respBody, "short_url") {
		t.Errorf("expected response to contain 'short_url' field, got %s", respBody)
	}
}

func TestBatchShorten_EmptyBatch(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, nil, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonBody := `[]`
	body := strings.NewReader(jsonBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/shorten/batch", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchShorten(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestBatchShorten_InvalidJSON(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, nil, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := strings.NewReader(`[{"correlation_id":}]`)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/shorten/batch", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchShorten(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestBatchShorten_DBError(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, &mockStorage{shouldFail: false}, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonBody := `[{"correlation_id":"1","original_url":"https://example.com"}]`
	body := strings.NewReader(jsonBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/shorten/batch", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchShorten(c)

	// DB fails, but falls back to disk storage successfully
	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

// для хендлера с URL пользователя
func TestGetUserURLs_Success(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, &mockStorage{shouldFail: false}, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Устанавливаем user_id в контекст
	c.Set("user_id", "test-user-123")
	c.Request = httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)

	handler.GetUserURLs(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	respBody := w.Body.String()
	if !strings.Contains(respBody, "short_url") {
		t.Errorf("expected response to contain 'short_url' field, got %s", respBody)
	}
	if !strings.Contains(respBody, "original_url") {
		t.Errorf("expected response to contain 'original_url' field, got %s", respBody)
	}
}

func TestGetUserURLs_NoUserID(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, &mockStorage{shouldFail: false}, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// без user_id
	c.Request = httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)

	handler.GetUserURLs(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestGetUserURLs_EmptyUserID(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, &mockStorage{shouldFail: false}, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// пустой user_id
	c.Set("user_id", "")
	c.Request = httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)

	handler.GetUserURLs(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestGetUserURLs_NoContent(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	// пстой список
	emptyStorage := &mockStorageEmpty{}

	svc := service.New(cache, emptyStorage, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Set("user_id", "test-user-123")
	c.Request = httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)

	handler.GetUserURLs(c)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestGetUserURLs_StorageError(t *testing.T) {
	testLogger := zap.NewNop().Sugar()

	cache := &mockCacheService{}

	svc := service.New(cache, &mockStorage{shouldFail: true}, testLogger)
	handler := NewURLHandler(svc, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Set("user_id", "test-user-123")
	c.Request = httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)

	handler.GetUserURLs(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// мок для тестирования пустого списка URL
type mockStorageEmpty struct{}

func (m *mockStorageEmpty) Insert(ctx context.Context, data model.Data) error {
	return nil
}

func (m *mockStorageEmpty) InsertBatch(ctx context.Context, dataList []model.Data) error {
	return nil
}

func (m *mockStorageEmpty) Select(ctx context.Context) ([]model.Data, error) {
	return []model.Data{}, nil
}

func (m *mockStorageEmpty) GetURLByOriginal(ctx context.Context, originalURL string) (string, error) {
	return "", errors.New("not found")
}

func (m *mockStorageEmpty) GetURLsByUser(ctx context.Context, userID string) ([]model.Data, error) {
	return []model.Data{}, nil
}

func (m *mockStorageEmpty) Ping(ctx context.Context) error {
	return nil
}
