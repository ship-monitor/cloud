package data

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
)

type OrgInvitationInput struct {
	OrganizationID uuid.UUID
	InviteeEmail   string
	ExpiresAt      time.Time
}

func CreateInvitation(inv OrgInvitationInput) (*domain.OrganizationInvitation, error) {
	invitation := domain.OrganizationInvitation{
		ID:             uuid.New(),
		OrganizationID: inv.OrganizationID,
		InviteeEmail:   inv.InviteeEmail,
		Status:         domain.InvStatusPending,
		CreatedAt:      time.Now(),
		ExpiresAt:      inv.ExpiresAt,
	}

	_, err := db.DB.NewInsert().Model(&invitation).Exec(context.Background())
	if err != nil {
		return nil, fmt.Errorf("insert invitation: %w", err)
	}

	return &invitation, nil
}

func GetInvitationByID(ctx context.Context, id uuid.UUID) (*domain.OrganizationInvitation, error) {
	var inv domain.OrganizationInvitation

	err := db.DB.NewSelect().
		Model(&inv).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select invitations: %w", err)
	}

	return &inv, nil
}

func HasPendingInvitation(orgID uuid.UUID, email string) (bool, error) {
	exists, err := db.DB.NewSelect().
		Model((*domain.OrganizationInvitation)(nil)).
		Where("invitee_email = ?", email).
		Where("status = ?", domain.InvStatusPending).
		Where("organization_id = ?", orgID).
		Exists(context.Background())
	if err != nil {
		return false, fmt.Errorf("select invitations: %w", err)
	}

	return exists, nil
}

func ListInvitationsForUser(email string) ([]domain.OrganizationInvitation, error) {
	var invs []domain.OrganizationInvitation

	err := db.DB.NewSelect().
		Model(&invs).
		Where("invitee_email = ? AND status = ?", email, domain.InvStatusPending).
		Order("created_at DESC").
		Scan(context.Background())
	if err != nil {
		return nil, fmt.Errorf("select invitations: %w", err)
	}

	return invs, nil
}

func ListInvitationsForOrg(orgID uuid.UUID) ([]domain.OrganizationInvitation, error) {
	var invs []domain.OrganizationInvitation

	err := db.DB.NewSelect().
		Model(&invs).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Scan(context.Background())
	if err != nil {
		return nil, fmt.Errorf("select invitations: %w", err)
	}

	return invs, nil
}

func UpdateInvitationStatus(id uuid.UUID, status domain.InvitationStatus) error {
	_, err := db.DB.NewUpdate().
		Model((*domain.OrganizationInvitation)(nil)).
		Set("status = ?", status).
		Where("id = ?", id).
		Exec(context.Background())
	if err != nil {
		return fmt.Errorf("update invitation: %w", err)
	}

	return nil
}
