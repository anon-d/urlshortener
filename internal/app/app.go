package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	config "github.com/anon-d/urlshortener/internal/config/flag"
	"github.com/anon-d/urlshortener/internal/handler"
	"github.com/anon-d/urlshortener/internal/logger"
	"github.com/anon-d/urlshortener/internal/middleware"
	"github.com/anon-d/urlshortener/internal/repository/cache"
	"github.com/anon-d/urlshortener/internal/repository/db/postgres"
	"github.com/anon-d/urlshortener/internal/repository/local"
	"github.com/anon-d/urlshortener/internal/service"
	serviceCache "github.com/anon-d/urlshortener/internal/service/cache"
	serviceDB "github.com/anon-d/urlshortener/internal/service/db"
	serviceLocal "github.com/anon-d/urlshortener/internal/service/local"
)

type App struct {
	server     *http.Server
	router     *gin.Engine
	urlHandler *handler.URLHandler
}

func New() (*App, error) {

	cfg := config.NewServerConfig()

	log, err := logger.New()
	if err != nil {
		return &App{}, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// database connection (optional)
	var db *postgres.Repository
	var dbService *serviceDB.DBService

	// Try to connect to database if DSN is provided
	if cfg.DSN != "" {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		var err error
		db, err = postgres.NewRepository(ctx, cfg.DSN, log)
		if err != nil {
			log.Warnw("Failed to connect to database, using file storage", "error", err)
			db = nil
		} else if err := db.Ping(ctx); err != nil {
			log.Warnw("Failed to ping database, using file storage", "error", err)
			db = nil
		} else {
			// If all correct
			dbService = serviceDB.New(db)
		}
	}

	// local storage
	localStorage := local.New(cfg.File, log)
	fileService := serviceLocal.New(localStorage)

	// Initialize cache from db (if available) or from file
	cache := cache.New(db, localStorage)
	cacheService := serviceCache.New(cache)

	service := service.New(cacheService, fileService, dbService, log)

	urlHandler := handler.NewURLHandler(service, cfg.AddrURL, log)

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
	router.Use(middleware.GlobalMiddleware(log)...)

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
	a.router.GET("/ping", a.urlHandler.PingDB)
	a.router.POST("/api/shorten/batch", a.urlHandler.BatchShorten)
	a.router.NoMethod(a.urlHandler.NotAllowed)
	a.router.NoRoute(a.urlHandler.NotFound)

}

func (a *App) Run() error {
	return a.server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) {
}
