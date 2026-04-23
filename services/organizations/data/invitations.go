package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
)

// InvitationStatus represents the state of an invitation.
type InvitationStatus string

const (
	StatusPending  InvitationStatus = "pending"
	StatusAccepted InvitationStatus = "accepted"
	StatusDeclined InvitationStatus = "declined"
)

// OrganizationInvitation holds the invitation record.
type OrganizationInvitation struct {
	*bun.BaseModel `bun:"table:organization_invitations"`

	ID             uuid.UUID        `bun:",pk,type:varchar" json:"id"`
	OrganizationID uuid.UUID        `bun:",notnull,type:varchar" json:"organizationId"`
	InviteeEmail   string           `bun:",notnull" json:"inviteeEmail"`
	Token          string           `bun:",notnull,unique" json:"token"`
	Status         InvitationStatus `bun:",notnull" json:"status"`
	CreatedAt      time.Time        `bun:",nullzero,notnull" json:"createdAt"`
	ExpiresAt      time.Time        `bun:",nullzero,notnull" json:"expiresAt"`
}

// CreateInvitation inserts a new invitation record.
func CreateInvitation(inv OrgInvitationInput) (*OrganizationInvitation, error) {
	invitation := OrganizationInvitation{
		OrganizationID: inv.OrganizationID,
		InviteeEmail:   inv.InviteeEmail,
		Token:          inv.Token,
		Status:         StatusPending,
		CreatedAt:      time.Now(),
		ExpiresAt:      inv.ExpiresAt,
	}
	_, err := db.DB.NewInsert().Model(&invitation).Exec(context.Background())
	if err != nil {
		return nil, err
	}
	return &invitation, nil
}

// OrgInvitationInput is the minimal data needed to create an invitation.
type OrgInvitationInput struct {
	OrganizationID uuid.UUID
	InviteeEmail   string
	Token          string
	ExpiresAt      time.Time
}

// GetInvitationByToken retrieves an invitation by its token.
func GetInvitationByToken(token string) (*OrganizationInvitation, error) {
	var inv OrganizationInvitation
	err := db.DB.NewSelect().
		Model(&inv).
		Where("token = ?", token).
		Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// ListPendingInvitations returns all pending invitations for an organization.
func ListPendingInvitations(orgID uuid.UUID) ([]OrganizationInvitation, error) {
	var invs []OrganizationInvitation
	err := db.DB.NewSelect().
		Model(&invs).
		Where("organization_id = ? AND status = ?", orgID, StatusPending).
		Scan(context.Background())
	return invs, err
}

// UpdateInvitationStatus changes the status of an invitation.
func UpdateInvitationStatus(token string, status InvitationStatus) error {
	_, err := db.DB.NewUpdate().
		Model((*OrganizationInvitation)(nil)).
		Set("status = ?", status).
		Where("token = ?", token).
		Exec(context.Background())
	return err
}
