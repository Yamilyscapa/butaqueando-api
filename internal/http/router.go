package apihttp

import (
	"net/http"

	"github.com/butaqueando/api/internal/http/middleware"
	"github.com/butaqueando/api/internal/modules/auth"
	"github.com/butaqueando/api/internal/modules/follows"
	"github.com/butaqueando/api/internal/modules/health"
	"github.com/butaqueando/api/internal/modules/plays"
	"github.com/butaqueando/api/internal/modules/users"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Dependencies struct {
	DB          *gorm.DB
	TokenConfig auth.TokenConfig
}

func NewRouter(deps Dependencies) *gin.Engine {
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(
		gin.Logger(),
		middleware.RequestID(),
		middleware.ErrorEnvelope(),
		middleware.Recovery(),
	)

	router.NoRoute(func(c *gin.Context) {
		httpx.AbortError(c, http.StatusNotFound, "NOT_FOUND", "route not found", nil)
	})

	router.NoMethod(func(c *gin.Context) {
		httpx.AbortError(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
	})

	router.GET("/", func(c *gin.Context) {
		httpx.WriteData(c, http.StatusOK, gin.H{
			"name":   "butaqueando-api",
			"status": "ok",
		})
	})

	v1 := router.Group("/v1")

	health.RegisterRoutes(v1, health.Dependencies{DB: deps.DB})
	auth.RegisterRoutes(v1, auth.Dependencies{DB: deps.DB, TokenConfig: deps.TokenConfig})
	users.RegisterRoutes(v1, users.Dependencies{DB: deps.DB})
	plays.RegisterRoutes(v1, plays.Dependencies{DB: deps.DB})
	follows.RegisterRoutes(v1, follows.Dependencies{DB: deps.DB})

	return router
}
