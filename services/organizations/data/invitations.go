package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
)

type InvitationStatus string

const (
	StatusPending  InvitationStatus = "pending"
	StatusAccepted InvitationStatus = "accepted"
	StatusDeclined InvitationStatus = "declined"
)

type OrganizationInvitation struct {
	*bun.BaseModel `bun:"table:organization_invitations"`

	ID             uuid.UUID        `bun:",pk,type:varchar" json:"id"`
	OrganizationID uuid.UUID        `bun:",notnull,type:varchar" json:"organizationId"`
	InviteeEmail   string           `bun:",notnull" json:"inviteeEmail"`
	Status         InvitationStatus `bun:",notnull" json:"status"`
	CreatedAt      time.Time        `bun:",nullzero,notnull" json:"createdAt"`
	ExpiresAt      time.Time        `bun:",nullzero,notnull" json:"expiresAt"`
}

type OrgInvitationInput struct {
	OrganizationID uuid.UUID
	InviteeEmail   string
	ExpiresAt      time.Time
}

func CreateInvitation(inv OrgInvitationInput) (*OrganizationInvitation, error) {
	invitation := OrganizationInvitation{
		ID:             uuid.New(),
		OrganizationID: inv.OrganizationID,
		InviteeEmail:   inv.InviteeEmail,
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

func GetInvitationByID(id uuid.UUID) (*OrganizationInvitation, error) {
	var inv OrganizationInvitation
	err := db.DB.NewSelect().
		Model(&inv).
		Where("id = ?", id).
		Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func HasPendingInvitation(orgID uuid.UUID, email string) (bool, error) {
	return db.DB.NewSelect().
		Model((*OrganizationInvitation)(nil)).
		Where("organization_id = ? AND invitee_email = ? AND status = ?", orgID, email, StatusPending).
		Exists(context.Background())
}

func ListInvitationsForUser(email string) ([]OrganizationInvitation, error) {
	var invs []OrganizationInvitation
	err := db.DB.NewSelect().
		Model(&invs).
		Where("invitee_email = ? AND status = ?", email, StatusPending).
		Order("created_at DESC").
		Scan(context.Background())
	return invs, err
}

func ListInvitationsForOrg(orgID uuid.UUID) ([]OrganizationInvitation, error) {
	var invs []OrganizationInvitation
	err := db.DB.NewSelect().
		Model(&invs).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Scan(context.Background())
	return invs, err
}

func UpdateInvitationStatus(id uuid.UUID, status InvitationStatus) error {
	_, err := db.DB.NewUpdate().
		Model((*OrganizationInvitation)(nil)).
		Set("status = ?", status).
		Where("id = ?", id).
		Exec(context.Background())
	return err
}
