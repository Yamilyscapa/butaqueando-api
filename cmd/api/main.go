package main

import (
	"log"

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

	log.Printf("server running on http://localhost:%s", application.Config.Port)
	if err := application.Router.Run(":" + application.Config.Port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
