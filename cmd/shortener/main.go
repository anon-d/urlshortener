package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anon-d/urlshortener/internal/app"
)

func main() {
	application, err := app.New()
	if err != nil {
		log.Fatalf("Error initializing application.\n%s", err.Error())
	}
	application.SetupRoutes()

	go func() {
		if err := application.Run(); err != nil {
			if err.Error() != "http: Server closed" {
				log.Printf("Error running application: %s", err.Error())
			}
		}
	}()

	out := make(chan os.Signal, 1)
	signal.Notify(out, syscall.SIGINT, syscall.SIGTERM)
	<-out
	log.Print("Shutdown process is started...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	shut := make(chan struct{})
	go func() {
		application.Shutdown(ctx)
		close(shut)
	}()
	select {
	case <-shut:
		log.Print("Shutdown process is completed!")
	case <-ctx.Done():
		log.Print("Shutdown process timeout - exiting anyway")
	}
}
