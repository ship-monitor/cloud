package services

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/go-playground/validator/v10"
)

var _ EmailSender = (*EmailService)(nil)

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
	if err := validator.New().Struct(conf); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &EmailService{
		auth: smtp.PlainAuth("", conf.AuthEmail, conf.AuthPassword, conf.SMTPHost),
	}, nil
}

// SendEmail implements [EmailSender].
func (e *EmailService) SendEmail(ctx context.Context, email IEmail) error {
	msg := email.WriteEmail(e.conf.SenderName, e.conf.AuthEmail)

	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", e.conf.SMTPHost, e.conf.SMTPPort),
		e.auth,
		e.conf.AuthEmail,
		[]string{email.ReceiverEmail()}, msg,
	)
	if err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}

	return nil
}
