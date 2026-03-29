package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/anon-d/urlshortener/internal/handler"
	"github.com/anon-d/urlshortener/internal/model"
	"github.com/anon-d/urlshortener/internal/service"
	"github.com/anon-d/urlshortener/internal/worker"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// mockCacheServiceEx — мок кэша для примеров.
type mockCacheServiceEx struct {
	data map[string]string
}

func (m *mockCacheServiceEx) Set(data *model.Data) {
	m.data[data.ID] = data.OriginalURL
}

func (m *mockCacheServiceEx) Get(id string) (string, bool) {
	val, ok := m.data[id]
	return val, ok
}

func (m *mockCacheServiceEx) Self() []model.Data { return nil }

// mockStorageEx — мок хранилища для примеров.
type mockStorageEx struct{}

func (m *mockStorageEx) Insert(ctx context.Context, data model.Data) error     { return nil }
func (m *mockStorageEx) InsertBatch(ctx context.Context, d []model.Data) error { return nil }
func (m *mockStorageEx) Select(ctx context.Context) ([]model.Data, error)      { return nil, nil }
func (m *mockStorageEx) GetURLByOriginal(ctx context.Context, u string) (string, error) {
	return "", errors.New("not found")
}
func (m *mockStorageEx) GetURLsByUser(ctx context.Context, userID string) ([]model.Data, error) {
	return []model.Data{
		{ShortURL: "abc123", OriginalURL: "https://example.com"},
	}, nil
}
func (m *mockStorageEx) GetURLByShortURL(ctx context.Context, shortURL string) (model.Data, error) {
	return model.Data{ShortURL: shortURL, OriginalURL: "https://example.com"}, nil
}
func (m *mockStorageEx) BatchMarkAsDeleted(ctx context.Context, r []worker.DeleteRequest) error {
	return nil
}
func (m *mockStorageEx) Ping(ctx context.Context) error { return nil }

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestHandler() *handler.URLHandler {
	logger := zap.NewNop().Sugar()
	cache := &mockCacheServiceEx{data: make(map[string]string)}
	svc := service.New(cache, &mockStorageEx{}, logger)
	deleteChan := make(chan handler.DeleteRequest, 100)
	return handler.NewURLHandler(svc, "http://localhost:8080", logger, deleteChan, nil)
}

// ExampleURLHandler_PostURL демонстрирует сокращение URL через POST /.
// Оригинальный URL передаётся в теле запроса как plain-text.
// В ответ возвращается короткий URL со статусом 201 Created.
func ExampleURLHandler_PostURL() {
	h := newTestHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))

	h.PostURL(c)

	fmt.Println("Status:", w.Code)
	fmt.Println("Body contains base URL:", strings.Contains(w.Body.String(), "http://localhost:8080"))
	// Output:
	// Status: 201
	// Body contains base URL: true
}

// ExampleURLHandler_Shorten демонстрирует сокращение URL через POST /api/shorten.
// Оригинальный URL передаётся в JSON-теле по ключу "url".
// В ответ возвращается JSON с полем "result", содержащим короткий URL.
func ExampleURLHandler_Shorten() {
	h := newTestHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/shorten",
		strings.NewReader(`{"url":"https://example.com"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Shorten(c)

	fmt.Println("Status:", w.Code)

	var resp handler.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	fmt.Println("Result contains base URL:", strings.Contains(resp.Result, "http://localhost:8080"))
	// Output:
	// Status: 201
	// Result contains base URL: true
}

// ExampleURLHandler_GetURL демонстрирует редирект по короткому URL через GET /:id.
// Возвращается 307 Temporary Redirect с заголовком Location.
func ExampleURLHandler_GetURL() {
	logger := zap.NewNop().Sugar()
	cache := &mockCacheServiceEx{data: map[string]string{
		"abc123": "https://example.com",
	}}
	svc := service.New(cache, &mockStorageEx{}, logger)
	deleteChan := make(chan handler.DeleteRequest, 100)
	h := handler.NewURLHandler(svc, "http://localhost:8080", logger, deleteChan, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/abc123", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc123"}}

	h.GetURL(c)

	fmt.Println("Status:", w.Code)
	fmt.Println("Location:", w.Header().Get("Location"))
	// Output:
	// Status: 307
	// Location: https://example.com
}

// ExampleURLHandler_BatchShorten демонстрирует пакетное сокращение URL
// через POST /api/shorten/batch.
// В теле запроса передаётся JSON-массив объектов с correlation_id и original_url.
// В ответе — массив объектов с correlation_id и short_url.
func ExampleURLHandler_BatchShorten() {
	h := newTestHandler()

	body := `[{"correlation_id":"1","original_url":"https://example.com"}]`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.BatchShorten(c)

	fmt.Println("Status:", w.Code)

	var resp []handler.ItemBatchResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	fmt.Println("Items:", len(resp))
	fmt.Println("Has correlation_id:", resp[0].CorrelationID == "1")
	// Output:
	// Status: 201
	// Items: 1
	// Has correlation_id: true
}

// ExampleURLHandler_GetUserURLs демонстрирует получение всех URL пользователя
// через GET /api/user/urls.
// Требует авторизации (user_id в контексте Gin).
func ExampleURLHandler_GetUserURLs() {
	h := newTestHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", "test-user")
	c.Request = httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)

	h.GetUserURLs(c)

	fmt.Println("Status:", w.Code)
	fmt.Println("Body contains short_url:", strings.Contains(w.Body.String(), "short_url"))
	// Output:
	// Status: 200
	// Body contains short_url: true
}

// ExampleURLHandler_DeleteURLs демонстрирует асинхронное удаление URL
// через DELETE /api/user/urls.
// В теле запроса передаётся JSON-массив коротких URL.
// Возвращается 202 Accepted — удаление происходит асинхронно.
func ExampleURLHandler_DeleteURLs() {
	h := newTestHandler()

	body := `["abc123","def456"]`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", "test-user")
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.DeleteURLs(c)

	fmt.Println("Status:", w.Code)
	// Output:
	// Status: 200
}
