package media

import (
	"time"

	"github.com/butaqueando/api/internal/shared/storage"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const BasePath = "/media"

type Dependencies struct {
	DB             *gorm.DB
	PlaysStorage   storage.Client
	UsersStorage   storage.Client
	DownloadURLTTL time.Duration
}

func RegisterRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	repo := NewRepository(deps.DB)
	service := NewService(repo, deps.PlaysStorage, deps.UsersStorage, deps.DownloadURLTTL)
	handler := NewHandler(service)

	group := v1.Group(BasePath)
	group.GET("/plays/:playId/:mediaId", handler.GetPlayMedia)
	group.GET("/users/:userId/avatar", handler.GetUserAvatar)
}
