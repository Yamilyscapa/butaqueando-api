package email

import (
	"context"
	"fmt"
	"strings"

	resend "github.com/resend/resend-go/v3"
)

const defaultLoginCodeTemplateID = "login-code"

type ResendSender struct {
	client     *resend.Client
	from       string
	templateID string
}

func NewResendSender(apiKey string, from string, templateID string) (*ResendSender, error) {
	trimmedAPIKey := strings.TrimSpace(apiKey)
	trimmedFrom := strings.TrimSpace(from)
	trimmedTemplateID := strings.TrimSpace(templateID)

	if trimmedAPIKey == "" {
		return nil, fmt.Errorf("resend api key is required")
	}

	if trimmedFrom == "" {
		return nil, fmt.Errorf("resend from email is required")
	}

	if trimmedTemplateID == "" {
		trimmedTemplateID = defaultLoginCodeTemplateID
	}

	return &ResendSender{
		client:     resend.NewClient(trimmedAPIKey),
		from:       trimmedFrom,
		templateID: trimmedTemplateID,
	}, nil
}

func (s *ResendSender) SendVerificationEmail(ctx context.Context, input VerificationEmailInput) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("email sender is not configured")
	}

	toEmail := strings.TrimSpace(input.ToEmail)
	if toEmail == "" {
		return fmt.Errorf("recipient email is required")
	}

	redirect := strings.TrimSpace(input.Redirect)
	if redirect == "" {
		return fmt.Errorf("verification redirect is required")
	}

	req := &resend.SendEmailRequest{
		From: s.from,
		To:   []string{toEmail},
		Template: &resend.EmailTemplate{
			Id: s.templateID,
			Variables: map[string]interface{}{
				"redirect": redirect,
			},
		},
	}

	options := &resend.SendEmailOptions{}
	if key := strings.TrimSpace(input.IdempotencyKey); key != "" {
		options.IdempotencyKey = key
	}

	_, err := s.client.Emails.SendWithOptions(ctx, req, options)
	if err != nil {
		return fmt.Errorf("resend send email: %w", err)
	}

	return nil
}
