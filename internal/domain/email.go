package domain

import (
	"context"

	"github.com/ship-monitor/cloud/pkg/email"
)

type EmailSender interface {
	SendEmail(ctx context.Context, e email.Email) error
}
