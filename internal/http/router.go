package apihttp

import (
	"net/http"
	"time"

	"github.com/butaqueando/api/internal/http/middleware"
	"github.com/butaqueando/api/internal/modules/auth"
	"github.com/butaqueando/api/internal/modules/follows"
	"github.com/butaqueando/api/internal/modules/health"
	"github.com/butaqueando/api/internal/modules/media"
	"github.com/butaqueando/api/internal/modules/plays"
	"github.com/butaqueando/api/internal/modules/users"
	"github.com/butaqueando/api/internal/shared/cache"
	sharedemail "github.com/butaqueando/api/internal/shared/email"
	"github.com/butaqueando/api/internal/shared/httpx"
	"github.com/butaqueando/api/internal/shared/storage"
	"github.com/butaqueando/api/internal/shared/worker"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Dependencies struct {
	DB                        *gorm.DB
	TokenConfig               auth.TokenConfig
	EmailVerificationRequired bool
	ExposeVerificationToken   bool
	VerificationEmailSender   sharedemail.Sender
	EmailVerificationRedirect string
	PasswordResetRedirect     string
	PasswordResetTokenTTL     time.Duration
	PlaysStorage              storage.Client
	UsersStorage              storage.Client
	S3UploadURLTTL            time.Duration
	S3DownloadURLTTL          time.Duration
	S3MaxImageBytes           int64
	ImageQueue                *worker.Queue
	ImageOptimizationEnabled  bool
	ImageWebPQuality          int
	MediaCache                cache.Client
}

func NewRouter(deps Dependencies) *gin.Engine {
	accessTokenManager := auth.NewTokenManager(deps.TokenConfig)
	accessTokenParser := func(rawToken string) (middleware.AccessTokenClaims, error) {
		claims, err := accessTokenManager.ParseAccessToken(rawToken)
		if err != nil {
			return middleware.AccessTokenClaims{}, err
		}

		return middleware.AccessTokenClaims{UserID: claims.UserID, Role: claims.Role}, nil
	}

	playsStorage := deps.PlaysStorage
	if playsStorage == nil {
		playsStorage = storage.NoopClient{}
	}

	usersStorage := deps.UsersStorage
	if usersStorage == nil {
		usersStorage = storage.NoopClient{}
	}

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
	auth.RegisterRoutes(v1, auth.Dependencies{
		DB:                        deps.DB,
		TokenConfig:               deps.TokenConfig,
		EmailVerificationRequired: deps.EmailVerificationRequired,
		ExposeVerificationToken:   deps.ExposeVerificationToken,
		VerificationEmailSender:   deps.VerificationEmailSender,
		EmailVerificationRedirect: deps.EmailVerificationRedirect,
		PasswordResetRedirect:     deps.PasswordResetRedirect,
		PasswordResetTokenTTL:     deps.PasswordResetTokenTTL,
	})
	users.RegisterRoutes(v1, users.Dependencies{
		DB:                deps.DB,
		AccessTokenParser: accessTokenParser,
		MediaStorage:      usersStorage,
		UploadURLTTL:      deps.S3UploadURLTTL,
		MaxImageBytes:     deps.S3MaxImageBytes,
		ImageQueue:        deps.ImageQueue,
		OptimizeImages:    deps.ImageOptimizationEnabled,
		WebPQuality:       deps.ImageWebPQuality,
	})
	plays.RegisterRoutes(v1, plays.Dependencies{
		DB:                deps.DB,
		AccessTokenParser: accessTokenParser,
		MediaStorage:      playsStorage,
		UploadURLTTL:      deps.S3UploadURLTTL,
		MaxImageBytes:     deps.S3MaxImageBytes,
		ImageQueue:        deps.ImageQueue,
		OptimizeImages:    deps.ImageOptimizationEnabled,
		WebPQuality:       deps.ImageWebPQuality,
		Cache:             deps.MediaCache,
	})
	media.RegisterRoutes(v1, media.Dependencies{
		DB:             deps.DB,
		PlaysStorage:   playsStorage,
		UsersStorage:   usersStorage,
		DownloadURLTTL: deps.S3DownloadURLTTL,
		Cache:          deps.MediaCache,
	})
	follows.RegisterRoutes(v1, follows.Dependencies{DB: deps.DB, AccessTokenParser: accessTokenParser})

	internalGroup := router.Group("/internal")
	internalGroup.GET("/metrics", func(c *gin.Context) {
		httpx.WriteData(c, http.StatusOK, gin.H{"cache": cache.Snapshot()})
	})

	return router
}
