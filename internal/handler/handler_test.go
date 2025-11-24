package handler

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anon-d/urlshortener/internal/service/url"
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

	body := strings.NewReader("https://example.com")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	handler.PostURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(respBody), "abc123") {
		t.Errorf("expected response to contain 'abc123', got %s", string(respBody))
	}
}

func TestPostURL_EmptyBody(t *testing.T) {
	store := &mockStore{}
	urlService := url.NewURLService(store)
	handler := NewURLHandler(urlService)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	handler.PostURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
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

	body := strings.NewReader("https://example.com")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	handler.PostURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
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

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	req.SetPathValue("id", "abc123")
	w := httptest.NewRecorder()

	handler.GetURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Errorf("expected status %d, got %d", http.StatusTemporaryRedirect, resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "https://example.com" {
		t.Errorf("expected Location header 'https://example.com', got %s", location)
	}
}

func TestGetURL_NotFound(t *testing.T) {
	store := &mockStore{
		getURLFunc: func(shortURL string) (string, error) {
			return "", errors.New("not found")
		},
	}
	urlService := url.NewURLService(store)
	handler := NewURLHandler(urlService)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	handler.GetURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
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

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetPathValue("id", "")
	w := httptest.NewRecorder()

	handler.GetURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}
