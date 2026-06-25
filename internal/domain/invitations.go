package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type InvitationStatus string

const (
	InvStatusPending  InvitationStatus = "pending"
	InvStatusAccepted InvitationStatus = "accepted"
	InvStatusDeclined InvitationStatus = "declined"
)

type OrganizationInvitation struct {
	*bun.BaseModel `bun:"table:organization_invitations"`

	ID             uuid.UUID        `bun:",pk,type:varchar"      json:"id"`
	OrganizationID uuid.UUID        `bun:",notnull,type:varchar" json:"organizationId"`
	InviteeEmail   string           `bun:",notnull"              json:"inviteeEmail"`
	Status         InvitationStatus `bun:",notnull"              json:"status"`
	CreatedAt      time.Time        `bun:",nullzero,notnull"     json:"createdAt"`
	ExpiresAt      time.Time        `bun:",nullzero,notnull"     json:"expiresAt"`
}
