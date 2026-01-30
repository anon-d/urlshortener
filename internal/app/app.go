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
	"github.com/anon-d/urlshortener/internal/repository"
	"github.com/anon-d/urlshortener/internal/repository/cache"
	"github.com/anon-d/urlshortener/internal/repository/db/postgres"
	"github.com/anon-d/urlshortener/internal/repository/local"
	"github.com/anon-d/urlshortener/internal/service"
	serviceCache "github.com/anon-d/urlshortener/internal/service/cache"
	"github.com/anon-d/urlshortener/internal/worker"
)

type App struct {
	server       *http.Server
	router       *gin.Engine
	urlHandler   *handler.URLHandler
	deleteWorker *worker.DeleteWorker
}

func New() (*App, error) {

	cfg := config.NewServerConfig()

	log, err := logger.New()
	if err != nil {
		return &App{}, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize local storage
	localStorage := local.New(cfg.File, log)

	// Initialize cache
	cacheStorage := cache.New(nil, localStorage)
	cacheService := serviceCache.New(cacheStorage)

	// Initialize storage (database or local file)
	var storage repository.Storage
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if cfg.DSN != "" {
		db, err := postgres.NewRepository(ctx, cfg.DSN, log)
		if err != nil {
			log.Warnw("Failed to connect to database, using file storage", "error", err)
		} else if err := db.Ping(ctx); err != nil {
			log.Warnw("Failed to ping database, using file storage", "error", err)
		} else {
			storage = repository.NewDBAdapter(db)
		}
	}

	// Fallback to local file storage
	if storage == nil {
		storage = repository.NewLocalAdapter(localStorage)
	}

	// Load data into cache from storage
	if cacheData, err := storage.Select(ctx); err == nil {
		for _, item := range cacheData {
			cacheStorage.Set(item.ID, item.OriginalURL)
		}
	}

	svc := service.New(cacheService, storage, log)

	// worker
	deleteWorkerAdapter := &deleteStorageAdapter{storage: storage}
	deleteWorker := worker.NewDeleteWorker(
		deleteWorkerAdapter,
		100,                  // buffer size
		500*time.Millisecond, // flush interval
		log,
	)

	// Создаем динамическое количество каналов на основе конфига
	deleteChannels := make([]chan handler.DeleteRequest, cfg.DeleteWorkerCount)
	for i := 0; i < cfg.DeleteWorkerCount; i++ {
		deleteChannels[i] = make(chan handler.DeleteRequest, cfg.DeleteChannelSize)
		workerChan := convertDeleteChannel(deleteChannels[i])
		deleteWorker.AddChannel(workerChan)
	}
	deleteWorker.Start()

	// канал для handler
	urlHandler := handler.NewURLHandler(svc, cfg.AddrURL, log, deleteChannels[0])

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
	router.Use(middleware.AuthMiddleware(cfg.SecretKey))

	router.HandleMethodNotAllowed = true

	httpServer := &http.Server{
		Addr:    cfg.AddrServer,
		Handler: router,
	}

	return &App{
		server:       httpServer,
		router:       router,
		urlHandler:   urlHandler,
		deleteWorker: deleteWorker,
	}, nil
}

// deleteStorageAdapter адаптер для работы с Storage через интерфейс DeleteStorage
type deleteStorageAdapter struct {
	storage repository.Storage
}

func (d *deleteStorageAdapter) BatchMarkAsDeleted(ctx context.Context, requests []worker.DeleteRequest) error {
	return d.storage.BatchMarkAsDeleted(ctx, requests)
}

// handler.DeleteRequest -> worker.DeleteRequest
func convertDeleteChannel(in <-chan handler.DeleteRequest) <-chan worker.DeleteRequest {
	out := make(chan worker.DeleteRequest)
	go func() {
		for req := range in {
			out <- worker.DeleteRequest{
				UserID:   req.UserID,
				ShortURL: req.ShortURL,
			}
		}
		close(out)
	}()
	return out
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
	a.router.GET("/api/user/urls", a.urlHandler.GetUserURLs)
	a.router.DELETE("/api/user/urls", a.urlHandler.DeleteURLs)
	a.router.NoMethod(a.urlHandler.NotAllowed)
	a.router.NoRoute(a.urlHandler.NotFound)

}

func (a *App) Run() error {
	return a.server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) {
	if a.deleteWorker != nil {
		a.deleteWorker.Stop()
	}
	if err := a.server.Shutdown(ctx); err != nil {
		fmt.Println("Failed shutdown. error: ", err)
	}
}
