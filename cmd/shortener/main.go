package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/anon-d/urlshortener/internal/app"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func checkLinkVar(variable string) string {
	if variable == "" {
		return "N/A"
	}
	return variable
}

func main() {
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", checkLinkVar(buildVersion), checkLinkVar(buildDate), checkLinkVar(buildCommit))
	application, err := app.New()
	if err != nil {
		log.Fatalf("failed to initialize application (check config flags, DB connection, file permissions): %s", err.Error())
	}
	application.SetupRoutes()

	go func() {
		if err := application.Run(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.Printf("HTTP server exited with unexpected error: %s", err.Error())
			}
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	<-ctx.Done()
	stop() // освобождаем ресурсы сигнального механизма досрочно

	log.Print("Shutdown process is started...")

	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	shut := make(chan struct{})
	go func() {
		application.Shutdown(shutCtx)
		close(shut)
	}()
	select {
	case <-shut:
		log.Print("Shutdown process is completed!")
	case <-shutCtx.Done():
		log.Print("Shutdown process timeout - exiting anyway")
	}
}
