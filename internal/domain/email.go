package domain

import (
	"context"

	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/email"
)

type EmailSender interface {
	SendEmail(ctx context.Context, e email.Email) error
}
