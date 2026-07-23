package services

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
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

	Disabled bool
}

type EmailService struct {
	auth smtp.Auth
	conf EmailServiceConfig
}

func NewEmailService(vConf *viper.Viper) (*EmailService, error) {
	conf := EmailServiceConfig{
		SMTPHost:     vConf.GetString("email.smtp-host"),
		SMTPPort:     vConf.GetUint("email.smtp-port"),
		SenderName:   vConf.GetString("email.sender-name"),
		AuthEmail:    vConf.GetString("email.email"),
		AuthPassword: vConf.GetString("email.password"),
		Disabled:     vConf.GetBool("email.disabled"),
	}
	if conf.Disabled {
		return &EmailService{
			conf: conf,
		}, nil
	}

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
func (s *EmailService) SendEmail(
	ctx context.Context,
	message email.Email,
) error {
	if s.conf.Disabled {
		return nil
	}

	w := email.NewHTMLWriter(email.Sender{
		Email: s.conf.AuthEmail,
		Name:  s.conf.SenderName,
	})

	msg, err := w.Write(message)
	if err != nil {
		return fmt.Errorf("write email: %w", err)
	}

	err = smtp.SendMail(
		fmt.Sprintf("%s:%d", s.conf.SMTPHost, s.conf.SMTPPort),
		s.auth,
		s.conf.AuthEmail,
		[]string{message.To}, msg,
	)
	if err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}

	return nil
}
