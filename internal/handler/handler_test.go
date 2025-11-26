package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	err "github.com/anon-d/urlshortener/internal/error"
	"github.com/anon-d/urlshortener/internal/service/url"
	"github.com/gin-gonic/gin"
)

type mockStore struct {
	addURLFunc func(id string, longURL string) (string, error)
	getURLFunc func(shortURL string) (string, error)
}

func (m *mockStore) AddURL(id string, longURL string) (string, error) {
	if m.addURLFunc != nil {
		return m.addURLFunc(id, longURL)
	}
	return "", errors.New("not implemented")
}

func (m *mockStore) GetURL(shortURL string) (string, error) {
	if m.getURLFunc != nil {
		return m.getURLFunc(shortURL)
	}
	return "", errors.New("not implemented")
}

func TestPostURL_Success(t *testing.T) {
	store := &mockStore{
		addURLFunc: func(id string, longURL string) (string, error) {
			return "abc123", nil
		},
	}
	urlService := url.NewURLService(store)
	handler := NewURLHandler(urlService)

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
	if !strings.Contains(respBody, "abc123") {
		t.Errorf("expected response to contain 'abc123', got %s", respBody)
	}
}

func TestPostURL_EmptyBody(t *testing.T) {
	store := &mockStore{}
	urlService := url.NewURLService(store)
	handler := NewURLHandler(urlService)

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
	store := &mockStore{
		addURLFunc: func(id string, longURL string) (string, error) {
			return "", errors.New("storage error")
		},
	}
	urlService := url.NewURLService(store)
	handler := NewURLHandler(urlService)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := strings.NewReader("https://example.com")
	c.Request = httptest.NewRequest(http.MethodPost, "/", body)

	handler.PostURL(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetURL_Success(t *testing.T) {
	store := &mockStore{
		getURLFunc: func(shortURL string) (string, error) {
			if shortURL == "abc123" {
				return "https://example.com", nil
			}
			return "", errors.New("not found")
		},
	}
	urlService := url.NewURLService(store)
	handler := NewURLHandler(urlService)

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
	store := &mockStore{
		getURLFunc: func(shortURL string) (string, error) {
			return "", err.ErrNotFound
		},
	}
	urlService := url.NewURLService(store)
	handler := NewURLHandler(urlService)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	handler.GetURL(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGetURL_EmptyID(t *testing.T) {
	store := &mockStore{
		getURLFunc: func(shortURL string) (string, error) {
			if shortURL == "" {
				return "", errors.New("empty id")
			}
			return "", errors.New("not found")
		},
	}
	urlService := url.NewURLService(store)
	handler := NewURLHandler(urlService)

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
