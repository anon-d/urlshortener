package main

import (
	"github.com/gin-gonic/gin"

	"github.com/anon-d/urlshortener/internal/handler"
	"github.com/anon-d/urlshortener/internal/model"
	"github.com/anon-d/urlshortener/internal/service/url"
)

func main() {
	store := model.NewStore()
	urlService := url.NewURLService(store)
	urlHandler := handler.NewURLHandler(urlService)

	router := gin.Default()

	router.POST("/", urlHandler.PostURL)
	router.GET("/:id", urlHandler.GetURL)

	if err := router.Run(":8080"); err != nil {
		panic(err)
	}
}
