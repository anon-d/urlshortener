package app

import (
	"context"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	config "github.com/anon-d/urlshortener/internal/config/flag"
	"github.com/anon-d/urlshortener/internal/handler"
	"github.com/anon-d/urlshortener/internal/logger"
	"github.com/anon-d/urlshortener/internal/middleware"
	"github.com/anon-d/urlshortener/internal/model"
	service "github.com/anon-d/urlshortener/internal/service/url"
)

type App struct {
	server     *http.Server
	router     *gin.Engine
	urlHandler *handler.URLHandler
}

func New() (*App, error) {

	cfg := config.NewServerConfig()

	logger, err := logger.New()
	if err != nil {
		return &App{}, err
	}

	store := model.NewStore()
	urlService := service.NewURLService(store)
	urlHandler := handler.NewURLHandler(urlService, cfg.AddrURL)

	// init Gin and http
	if cfg.Env == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	router := gin.Default()

	// CORS middleware
	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	// Security headers
	router.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	})

	// middleware
	router.Use(middleware.RequestMiddleware(logger))
	router.Use(middleware.ResponseMiddleware(logger))

	router.HandleMethodNotAllowed = true

	httpServer := &http.Server{
		Addr:    cfg.AddrServer,
		Handler: router,
	}

	return &App{
		server:     httpServer,
		router:     router,
		urlHandler: urlHandler,
	}, nil
}

func (a *App) SetupRoutes() {
	maintenance := a.router.Group("/maintenance")
	{
		maintenance.GET("/health", a.urlHandler.HealthCheck)
	}

	a.router.POST("/", a.urlHandler.PostURL)
	a.router.GET("/:id", a.urlHandler.GetURL)
	a.router.POST("/api/shorten", a.urlHandler.Shorten)
	a.router.NoMethod(a.urlHandler.NotAllowed)
	a.router.NoRoute(a.urlHandler.NotFound)

}

func (a *App) Run() error {
	return a.server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) {
}
