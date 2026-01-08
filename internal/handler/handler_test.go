package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anon-d/urlshortener/internal/logger"
	"github.com/anon-d/urlshortener/internal/service/storage"
	dbMocks "github.com/anon-d/urlshortener/internal/service/storage/mocks"
	"github.com/anon-d/urlshortener/internal/service/url"
	"github.com/anon-d/urlshortener/internal/service/url/mocks"
	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestPostURL_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockURLStore(ctrl)
	store.EXPECT().AddURL(gomock.Any(), gomock.Any()).Return(nil)

	db := dbMocks.NewMockIDB(ctrl)

	testLogger := &logger.Logger{ZLog: zap.NewNop().Sugar()}
	urlService := url.NewURLService(store, testLogger)
	dbService := storage.NewDBService(db)
	handler := NewURLHandler(urlService, dbService, "http://localhost:8080", testLogger)

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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockURLStore(ctrl)

	db := dbMocks.NewMockIDB(ctrl)

	testLogger := &logger.Logger{ZLog: zap.NewNop().Sugar()}
	urlService := url.NewURLService(store, testLogger)
	dbService := storage.NewDBService(db)
	handler := NewURLHandler(urlService, dbService, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	handler.PostURL(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPostURL_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockURLStore(ctrl)
	store.EXPECT().AddURL(gomock.Any(), gomock.Any()).Return(errors.New("storage error"))

	db := dbMocks.NewMockIDB(ctrl)

	testLogger := &logger.Logger{ZLog: zap.NewNop().Sugar()}
	urlService := url.NewURLService(store, testLogger)
	dbService := storage.NewDBService(db)
	handler := NewURLHandler(urlService, dbService, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := strings.NewReader("https://example.com")
	c.Request = httptest.NewRequest(http.MethodPost, "/", body)

	handler.PostURL(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestGetURL_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockURLStore(ctrl)
	store.EXPECT().GetURL("abc123").Return("https://example.com", true)

	db := dbMocks.NewMockIDB(ctrl)

	testLogger := &logger.Logger{ZLog: zap.NewNop().Sugar()}
	urlService := url.NewURLService(store, testLogger)
	dbService := storage.NewDBService(db)
	handler := NewURLHandler(urlService, dbService, "http://localhost:8080", testLogger)

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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockURLStore(ctrl)
	store.EXPECT().GetURL("nonexistent").Return(nil, false)

	db := dbMocks.NewMockIDB(ctrl)

	testLogger := &logger.Logger{ZLog: zap.NewNop().Sugar()}
	urlService := url.NewURLService(store, testLogger)
	dbService := storage.NewDBService(db)
	handler := NewURLHandler(urlService, dbService, "http://localhost:8080", testLogger)

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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockURLStore(ctrl)
	// тут мока нет, т.к. проверяю прям в хендлере

	db := dbMocks.NewMockIDB(ctrl)

	testLogger := &logger.Logger{ZLog: zap.NewNop().Sugar()}
	urlService := url.NewURLService(store, testLogger)
	dbService := storage.NewDBService(db)
	handler := NewURLHandler(urlService, dbService, "http://localhost:8080", testLogger)

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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockURLStore(ctrl)
	store.EXPECT().AddURL(gomock.Any(), "https://example.com").Return(nil)

	db := dbMocks.NewMockIDB(ctrl)

	testLogger := &logger.Logger{ZLog: zap.NewNop().Sugar()}
	urlService := url.NewURLService(store, testLogger)
	dbService := storage.NewDBService(db)
	handler := NewURLHandler(urlService, dbService, "http://localhost:8080", testLogger)

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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockURLStore(ctrl)

	db := dbMocks.NewMockIDB(ctrl)

	testLogger := &logger.Logger{ZLog: zap.NewNop().Sugar()}
	urlService := url.NewURLService(store, testLogger)
	dbService := storage.NewDBService(db)
	handler := NewURLHandler(urlService, dbService, "http://localhost:8080", testLogger)

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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockURLStore(ctrl)

	db := dbMocks.NewMockIDB(ctrl)

	testLogger := &logger.Logger{ZLog: zap.NewNop().Sugar()}
	urlService := url.NewURLService(store, testLogger)
	dbService := storage.NewDBService(db)
	handler := NewURLHandler(urlService, dbService, "http://localhost:8080", testLogger)

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

func TestShorten_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockURLStore(ctrl)
	store.EXPECT().AddURL(gomock.Any(), "https://example.com").Return(errors.New("storage error"))

	db := dbMocks.NewMockIDB(ctrl)

	testLogger := &logger.Logger{ZLog: zap.NewNop().Sugar()}
	urlService := url.NewURLService(store, testLogger)
	dbService := storage.NewDBService(db)
	handler := NewURLHandler(urlService, dbService, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonBody := `{"url":"https://example.com"}`
	body := strings.NewReader(jsonBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/shorten", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Shorten(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestDBPing_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockURLStore(ctrl)

	db := dbMocks.NewMockIDB(ctrl)
	db.EXPECT().Ping(gomock.Any()).Return(errors.New("ping error"))

	testLogger := &logger.Logger{ZLog: zap.NewNop().Sugar()}
	urlService := url.NewURLService(store, testLogger)
	dbService := storage.NewDBService(db)
	handler := NewURLHandler(urlService, dbService, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodPost, "/ping", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.PingDB(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestDBPing_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockURLStore(ctrl)

	db := dbMocks.NewMockIDB(ctrl)
	db.EXPECT().Ping(gomock.Any()).Return(nil)

	testLogger := &logger.Logger{ZLog: zap.NewNop().Sugar()}
	urlService := url.NewURLService(store, testLogger)
	dbService := storage.NewDBService(db)
	handler := NewURLHandler(urlService, dbService, "http://localhost:8080", testLogger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodPost, "/ping", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.PingDB(c)
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
