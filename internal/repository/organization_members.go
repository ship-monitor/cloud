package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
)

func (r *OrganizationsRepo) AddMember(
	ctx context.Context,
	member *domain.OrganizationMember,
) error {
	_, err := r.db.NewInsert().Model(member).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert member: %w", err)
	}

	return nil
}

func (r *OrganizationsRepo) UpdateMemberRole(
	ctx context.Context,
	orgID, userID uuid.UUID,
	role domain.Role,
) error {
	_, err := r.db.NewUpdate().
		Model((*domain.OrganizationMember)(nil)).
		Set("role = ?", role).
		Where("organization_id = ? AND member_id = ?", orgID, userID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}

	return nil
}

func (r *OrganizationsRepo) RemoveMember(
	ctx context.Context,
	orgID, userID uuid.UUID,
) error {
	_, err := r.db.NewDelete().
		Model((*domain.OrganizationMember)(nil)).
		Where("organization_id = ? AND member_id = ?", orgID, userID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete member: %w", err)
	}

	return nil
}

func (r *OrganizationsRepo) GetMember(
	ctx context.Context,
	orgID, userID uuid.UUID,
) (*domain.OrganizationMember, error) {
	var member domain.OrganizationMember

	err := r.db.NewSelect().
		Model(&member).
		Where("organization_id = ? AND member_id = ?", orgID, userID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select member: %w", err)
	}

	return &member, nil
}

func (r *OrganizationsRepo) GetMembersWithUserInfo(
	ctx context.Context,
	orgID uuid.UUID,
) ([]domain.OrganizationMemberWithUser, error) {
	var members []domain.OrganizationMemberWithUser

	err := r.db.NewSelect().
		TableExpr("organization_members AS om").
		ColumnExpr("om.member_id, om.organization_id, om.role, om.joined_at, u.name, u.email").
		Join("JOIN users AS u ON u.id = om.member_id").
		Where("om.organization_id = ?", orgID).
		Order("om.joined_at ASC").
		Scan(ctx, &members)
	if err != nil {
		return nil, fmt.Errorf("select members: %w", err)
	}

	return members, nil
}

func (r *OrganizationsRepo) DeleteOrganization(
	ctx context.Context,
	id uuid.UUID,
) error {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewDelete().
			Model((*domain.OrganizationMember)(nil)).
			Where("organization_id = ?", id).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("delete organization members: %w", err)
		}

		_, err = tx.NewDelete().
			Model((*domain.Organization)(nil)).
			Where("id = ?", id).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("delete organization: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}

	return nil
}
