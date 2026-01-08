package app

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	config "github.com/anon-d/urlshortener/internal/config/flag"
	"github.com/anon-d/urlshortener/internal/handler"
	"github.com/anon-d/urlshortener/internal/logger"
	"github.com/anon-d/urlshortener/internal/middleware"
	"github.com/anon-d/urlshortener/internal/model"
	"github.com/anon-d/urlshortener/internal/repository/postgres"
	serviceDB "github.com/anon-d/urlshortener/internal/service/storage"
	serviceURL "github.com/anon-d/urlshortener/internal/service/url"
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

	// database connection (optional)
	var db *postgres.Repository
	var dbService *serviceDB.DBService
	if cfg.DSN != "" {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		var err error
		db, err = postgres.NewRepository(ctx, cfg.DSN, logger)
		if err != nil {
			logger.ZLog.Warnw("Failed to connect to database, using file storage", "error", err)
		} else if err := db.Ping(ctx); err != nil {
			logger.ZLog.Warnw("Failed to ping database, using file storage", "error", err)
			db = nil
		} else {
			dbService = serviceDB.NewDBService(db)
		}
	}

	fileSrv := model.NewFileStore(cfg.File, logger)
	store, err := model.NewStore(fileSrv, logger)
	if err != nil {
		return &App{}, err
	}

	urlService := serviceURL.NewURLService(store, logger)
	urlHandler := handler.NewURLHandler(urlService, dbService, cfg.AddrURL, logger)

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
	router.Use(middleware.GlobalMiddleware(logger)...)

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
	a.router.NoMethod(a.urlHandler.NotAllowed)
	a.router.NoRoute(a.urlHandler.NotFound)

}

func (a *App) Run() error {
	return a.server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) {
}
