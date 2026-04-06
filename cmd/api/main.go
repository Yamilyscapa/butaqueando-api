package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/butaqueando/api/internal/app"
)

func main() {
	application, err := app.Bootstrap()
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}

	defer func() {
		if closeErr := application.Close(); closeErr != nil {
			log.Printf("close database: %v", closeErr)
		}
	}()

	server := &http.Server{
		Addr:    ":" + application.Config.Port,
		Handler: application.Router,
	}

	go func() {
		log.Printf("server running on http://localhost:%s", application.Config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server stopped: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server shutdown failed: %v", err)
	}
}
