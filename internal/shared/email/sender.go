package email

import "context"

type VerificationEmailInput struct {
	ToEmail        string
	Redirect       string
	IdempotencyKey string
}

type Sender interface {
	SendVerificationEmail(ctx context.Context, input VerificationEmailInput) error
}

type NoopSender struct{}

func (NoopSender) SendVerificationEmail(_ context.Context, _ VerificationEmailInput) error {
	return nil
}
