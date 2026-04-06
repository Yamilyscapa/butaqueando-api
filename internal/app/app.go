package app

import (
	"database/sql"

	"github.com/butaqueando/api/internal/config"
	"github.com/butaqueando/api/internal/shared/worker"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Application struct {
	Config config.Config
	Router *gin.Engine
	DB     *gorm.DB
	SQLDB  *sql.DB
	Worker *worker.Queue
}

func (a *Application) Close() error {
	if a.Worker != nil {
		a.Worker.Stop()
	}

	if a.SQLDB == nil {
		return nil
	}

	return a.SQLDB.Close()
}
