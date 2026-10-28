package email

import (
	"context"
	"fmt"

	mailjet "github.com/mailjet/mailjet-apiv3-go/v4"
)

// MailjetSender implements usecase.EmailSender using the Mailjet
// transactional email API (Send API v3.1).
type MailjetSender struct {
	client      *mailjet.Client
	fromEmail   string
	fromName    string
	sandboxMode bool
}

// NewMailjetSender constructs a MailjetSender with the supplied credentials.
// Provide a non-empty baseURL to override the Mailjet endpoint (useful in tests).
func NewMailjetSender(apiKey, secretKey, fromEmail, fromName string, baseURL ...string) *MailjetSender {
	client := mailjet.NewMailjetClient(apiKey, secretKey, baseURL...)
	return &MailjetSender{
		client:    client,
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

// NewSandboxSender constructs a MailjetSender that sends all messages in
// sandbox mode (Mailjet validates the request but does not deliver the email).
// Intended for integration tests against the live Mailjet API.
func NewSandboxSender(apiKey, secretKey, fromEmail, fromName string) *MailjetSender {
	return &MailjetSender{
		client:      mailjet.NewMailjetClient(apiKey, secretKey),
		fromEmail:   fromEmail,
		fromName:    fromName,
		sandboxMode: true,
	}
}

// SendWelcomeEmail sends the welcome email (templates/welcome.html and
// welcome.txt, rendered by renderWelcome) to toEmail.
func (s *MailjetSender) SendWelcomeEmail(_ context.Context, toEmail, toName string) error {
	msg, err := renderWelcome(toName)
	if err != nil {
		return fmt.Errorf("email: render welcome: %w", err)
	}

	messagesInfo := []mailjet.InfoMessagesV31{
		{
			From: &mailjet.RecipientV31{
				Email: s.fromEmail,
				Name:  s.fromName,
			},
			To: &mailjet.RecipientsV31{
				mailjet.RecipientV31{
					Email: toEmail,
					Name:  toName,
				},
			},
			Subject:  msg.Subject,
			TextPart: msg.Text,
			HTMLPart: msg.HTML,
		},
	}

	req := &mailjet.MessagesV31{
		Info:        messagesInfo,
		SandBoxMode: s.sandboxMode,
	}

	if _, err := s.client.SendMailV31(req); err != nil {
		return fmt.Errorf("email: mailjet send: %w", err)
	}
	return nil
}
