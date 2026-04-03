package email

import "context"

type VerificationEmailInput struct {
	ToEmail        string
	Redirect       string
	IdempotencyKey string
}

type PasswordResetEmailInput struct {
	ToEmail        string
	Redirect       string
	IdempotencyKey string
}

type Sender interface {
	SendVerificationEmail(ctx context.Context, input VerificationEmailInput) error
	SendPasswordResetEmail(ctx context.Context, input PasswordResetEmailInput) error
}

type NoopSender struct{}

func (NoopSender) SendVerificationEmail(_ context.Context, _ VerificationEmailInput) error {
	return nil
}

func (NoopSender) SendPasswordResetEmail(_ context.Context, _ PasswordResetEmailInput) error {
	return nil
}
