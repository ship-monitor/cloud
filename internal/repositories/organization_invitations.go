package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
)

func (r *OrganizationsRepo) CreateInvitation(
	ctx context.Context,
	orgID uuid.UUID,
	inviteeEmail string,
	expiresAt time.Time,
) (*domain.OrganizationInvitation, error) {
	invitation := domain.OrganizationInvitation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		InviteeEmail:   inviteeEmail,
		Status:         domain.InvStatusPending,
		CreatedAt:      time.Now(),
		ExpiresAt:      expiresAt,
	}

	_, err := r.db.NewInsert().Model(&invitation).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("insert invitation: %w", err)
	}

	return &invitation, nil
}

func (r *OrganizationsRepo) GetInvitationByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.OrganizationInvitation, error) {
	var inv domain.OrganizationInvitation

	err := r.db.NewSelect().
		Model(&inv).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select invitation: %w", err)
	}

	return &inv, nil
}

func (r *OrganizationsRepo) HasPendingInvitation(
	ctx context.Context,
	orgID uuid.UUID,
	email string,
) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*domain.OrganizationInvitation)(nil)).
		Where("invitee_email = ?", email).
		Where("status = ?", domain.InvStatusPending).
		Where("organization_id = ?", orgID).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("select invitations: %w", err)
	}

	return exists, nil
}

func (r *OrganizationsRepo) ListInvitationsForUser(
	ctx context.Context,
	email string,
) ([]domain.OrganizationInvitation, error) {
	var invs []domain.OrganizationInvitation

	err := r.db.NewSelect().
		Model(&invs).
		Where("invitee_email = ? AND status = ?", email, domain.InvStatusPending).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select invitations: %w", err)
	}

	return invs, nil
}

func (r *OrganizationsRepo) ListInvitationsForOrg(
	ctx context.Context,
	orgID uuid.UUID,
) ([]domain.OrganizationInvitation, error) {
	var invs []domain.OrganizationInvitation

	err := r.db.NewSelect().
		Model(&invs).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select invitations: %w", err)
	}

	return invs, nil
}

func (r *OrganizationsRepo) UpdateInvitationStatus(
	ctx context.Context,
	id uuid.UUID,
	status domain.InvitationStatus,
) error {
	_, err := r.db.NewUpdate().
		Model((*domain.OrganizationInvitation)(nil)).
		Set("status = ?", status).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update invitation: %w", err)
	}

	return nil
}
