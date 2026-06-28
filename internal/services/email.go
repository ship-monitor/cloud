package services

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/go-playground/validator/v10"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/email"
)

var _ domain.EmailSender = (*EmailService)(nil)

type EmailServiceConfig struct {
	SMTPHost     string `validate:"required,hostname"`
	SMTPPort     uint   `validate:"required,port"`
	AuthEmail    string `validate:"required,email"`
	AuthPassword string `validate:"required"`

	SenderName string `validate:"required"`
}

type EmailService struct {
	auth smtp.Auth
	conf EmailServiceConfig
}

func NewEmailService(conf EmailServiceConfig) (*EmailService, error) {
	err := validator.New().Struct(conf)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &EmailService{
		auth: smtp.PlainAuth(
			"",
			conf.AuthEmail,
			conf.AuthPassword,
			conf.SMTPHost,
		),
		conf: conf,
	}, nil
}

// SendEmail implements [EmailSender].
func (s *EmailService) SendEmail(ctx context.Context, e email.Email) error {
	w := email.NewHTMLWriter(email.Sender{
		Email: s.conf.AuthEmail,
		Name:  s.conf.SenderName,
	})

	msg, err := w.Write(e)
	if err != nil {
		return fmt.Errorf("write email: %w", err)
	}

	err = smtp.SendMail(
		fmt.Sprintf("%s:%d", s.conf.SMTPHost, s.conf.SMTPPort),
		s.auth,
		s.conf.AuthEmail,
		[]string{e.To}, msg,
	)
	if err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}

	return nil
}
