package main

import (
	"net/http"

	"github.com/anon-d/urlshortener/internal/handler"
	"github.com/anon-d/urlshortener/internal/model"
	"github.com/anon-d/urlshortener/internal/service/url"
)

func main() {

	store := model.NewStore()
	urlService := url.NewURLService(store)
	urlHandler := handler.NewURLHandler(urlService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /", urlHandler.PostURL)
	mux.HandleFunc("GET /{id}", urlHandler.GetURL)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	})

	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
