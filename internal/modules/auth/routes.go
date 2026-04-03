package auth

import (
	"time"

	"github.com/butaqueando/api/internal/http/middleware"
	sharedemail "github.com/butaqueando/api/internal/shared/email"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const BasePath = "/auth"

type Dependencies struct {
	DB                        *gorm.DB
	TokenConfig               TokenConfig
	EmailVerificationRequired bool
	ExposeVerificationToken   bool
	VerificationEmailSender   sharedemail.Sender
	EmailVerificationRedirect string
	PasswordResetRedirect     string
	PasswordResetTokenTTL     time.Duration
}

func RegisterRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	repo := NewRepository(deps.DB)
	tokens := NewTokenManager(deps.TokenConfig)
	emailVerificationRequired := deps.EmailVerificationRequired
	service := NewService(repo, tokens, ServiceOptions{
		EmailVerificationRequired: &emailVerificationRequired,
		ExposeVerificationToken:   deps.ExposeVerificationToken,
		VerificationEmailSender:   deps.VerificationEmailSender,
		EmailVerificationRedirect: deps.EmailVerificationRedirect,
		PasswordResetRedirect:     deps.PasswordResetRedirect,
		PasswordResetTokenTTL:     deps.PasswordResetTokenTTL,
	})
	handler := NewHandler(service)
	group := v1.Group(BasePath)

	group.POST("/sign-up", middleware.AuthRateLimit(15, time.Minute), handler.SignUp)
	group.POST("/sign-in", middleware.AuthRateLimit(15, time.Minute), handler.SignIn)
	group.POST("/refresh", middleware.AuthRateLimit(30, time.Minute), handler.Refresh)
	group.POST("/sign-out", handler.SignOut)
	group.POST("/verify-email", middleware.AuthRateLimit(30, time.Minute), handler.VerifyEmail)
	group.POST("/resend-verification", middleware.AuthRateLimit(15, time.Minute), handler.ResendVerification)
	group.POST("/forgot-password", middleware.AuthRateLimit(15, time.Minute), handler.ForgotPassword)
	group.POST("/reset-password", middleware.AuthRateLimit(15, time.Minute), handler.ResetPassword)
}
