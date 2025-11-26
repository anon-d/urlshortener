package main

import (
	"flag"

	"github.com/gin-gonic/gin"

	config "github.com/anon-d/urlshortener/internal/config/flag"
	"github.com/anon-d/urlshortener/internal/handler"
	"github.com/anon-d/urlshortener/internal/model"
	"github.com/anon-d/urlshortener/internal/service/url"
)

func main() {
	addrServer := flag.String("a", ":8080", "address to listen on")
	addrUrl := flag.String("b", "http://localhost:8080", "base URL for short URLs")
	flag.Parse()
	cfg := config.NewServerConfig(*addrServer, *addrUrl)

	store := model.NewStore()
	urlService := url.NewURLService(store)
	urlHandler := handler.NewURLHandler(urlService, cfg.AddrUrl)

	router := gin.Default()

	router.POST("/", urlHandler.PostURL)
	router.GET("/:id", urlHandler.GetURL)

	if err := router.Run(cfg.AddrServer); err != nil {
		panic(err)
	}
}
