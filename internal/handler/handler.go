package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/anon-d/urlshortener/internal/service/url"
)

type URLHandler struct {
	URLService *url.URLService
}

func NewURLHandler(urlService *url.URLService) *URLHandler {
	return &URLHandler{
		URLService: urlService,
	}
}

func (u *URLHandler) PostURL(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, err := u.URLService.ShortenURL(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	shortURL := fmt.Sprintf("http://%s/%s", r.Host, id)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func (u *URLHandler) GetURL(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	urlLong, err := u.URLService.GetURL(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Location", urlLong)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
