package services

import (
	"context"

	"charm.land/log/v2"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/email"
)

var _ domain.EmailSender = (*DummyEmailService)(nil)

type DummyEmailService struct {
	logger *log.Logger
}

func NewDummyEmailService(logger *log.Logger) *DummyEmailService {
	return &DummyEmailService{
		logger: logger.WithPrefix("DUMMY email"),
	}
}

// SendEmail implements [domain.EmailSender].
func (d *DummyEmailService) SendEmail(
	ctx context.Context,
	e email.Email,
) error {
	d.logger.Info("Sending dummy email", "to", e.To, "subject", e.Subject)

	return nil
}
