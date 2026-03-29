package app

import (
	"database/sql"

	"github.com/butaqueando/api/internal/config"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Application struct {
	Config config.Config
	Router *gin.Engine
	DB     *gorm.DB
	SQLDB  *sql.DB
}

func (a *Application) Close() error {
	if a.SQLDB == nil {
		return nil
	}

	return a.SQLDB.Close()
}
