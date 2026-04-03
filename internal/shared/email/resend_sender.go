package email

import (
	"context"
	"fmt"
	"strings"

	resend "github.com/resend/resend-go/v3"
)

const (
	defaultVerificationTemplateID  = "login-code"
	defaultPasswordResetTemplateID = "reset-password"
)

type ResendSender struct {
	client                  *resend.Client
	from                    string
	verificationTemplateID  string
	passwordResetTemplateID string
}

func NewResendSender(apiKey string, from string, verificationTemplateID string, passwordResetTemplateID string) (*ResendSender, error) {
	trimmedAPIKey := strings.TrimSpace(apiKey)
	trimmedFrom := strings.TrimSpace(from)
	trimmedVerificationTemplateID := strings.TrimSpace(verificationTemplateID)
	trimmedPasswordResetTemplateID := strings.TrimSpace(passwordResetTemplateID)

	if trimmedAPIKey == "" {
		return nil, fmt.Errorf("resend api key is required")
	}

	if trimmedFrom == "" {
		return nil, fmt.Errorf("resend from email is required")
	}

	if trimmedVerificationTemplateID == "" {
		trimmedVerificationTemplateID = defaultVerificationTemplateID
	}

	if trimmedPasswordResetTemplateID == "" {
		trimmedPasswordResetTemplateID = defaultPasswordResetTemplateID
	}

	return &ResendSender{
		client:                  resend.NewClient(trimmedAPIKey),
		from:                    trimmedFrom,
		verificationTemplateID:  trimmedVerificationTemplateID,
		passwordResetTemplateID: trimmedPasswordResetTemplateID,
	}, nil
}

func (s *ResendSender) SendVerificationEmail(ctx context.Context, input VerificationEmailInput) error {
	return s.sendTemplateEmail(ctx, input.ToEmail, input.Redirect, input.IdempotencyKey, s.verificationTemplateID)
}

func (s *ResendSender) SendPasswordResetEmail(ctx context.Context, input PasswordResetEmailInput) error {
	return s.sendTemplateEmail(ctx, input.ToEmail, input.Redirect, input.IdempotencyKey, s.passwordResetTemplateID)
}

func (s *ResendSender) sendTemplateEmail(ctx context.Context, toEmail string, redirect string, idempotencyKey string, templateID string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("email sender is not configured")
	}

	toEmail = strings.TrimSpace(toEmail)
	if toEmail == "" {
		return fmt.Errorf("recipient email is required")
	}

	redirect = strings.TrimSpace(redirect)
	if redirect == "" {
		return fmt.Errorf("verification redirect is required")
	}

	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return fmt.Errorf("template id is required")
	}

	req := &resend.SendEmailRequest{
		From: s.from,
		To:   []string{toEmail},
		Template: &resend.EmailTemplate{
			Id: templateID,
			Variables: map[string]interface{}{
				"redirect": redirect,
			},
		},
	}

	options := &resend.SendEmailOptions{}
	if key := strings.TrimSpace(idempotencyKey); key != "" {
		options.IdempotencyKey = key
	}

	_, err := s.client.Emails.SendWithOptions(ctx, req, options)
	if err != nil {
		return fmt.Errorf("resend send email: %w", err)
	}

	return nil
}
