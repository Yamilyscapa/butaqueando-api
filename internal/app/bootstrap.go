package app

import (
	"fmt"

	"github.com/butaqueando/api/internal/config"
	"github.com/butaqueando/api/internal/database"
	apihttp "github.com/butaqueando/api/internal/http"
)

func Bootstrap() (*Application, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	db, sqlDB, err := database.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	router := apihttp.NewRouter(apihttp.Dependencies{DB: db})

	return &Application{
		Config: cfg,
		Router: router,
		DB:     db,
		SQLDB:  sqlDB,
	}, nil
}
