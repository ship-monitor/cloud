package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
)

type OrganizationsRepo struct {
	db *bun.DB
}

func New(db *sql.DB) *OrganizationsRepo {
	return &OrganizationsRepo{
		db: bun.NewDB(db, sqlitedialect.New()),
	}
}

func (r *OrganizationsRepo) CreateOrganization(ctx context.Context, name string, creatorID uuid.UUID) (uuid.UUID, error) {

	orgID := uuid.New()
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return errors.Join(
			insertOrganization(ctx, tx, &data.Organization{
				ID:        orgID,
				Name:      name,
				CreatorID: creatorID,
				CreatedAt: time.Now(),
			}),
			insertMember(ctx, tx, &data.OrganizationMember{
				MemberID:       creatorID,
				OrganizationID: orgID,
				JoinedAt:       time.Now(),
				Role:           data.RoleOwner,
			}))
	})
	if err != nil {
		return orgID, fmt.Errorf("create organization: %w", err)
	}

	return orgID, nil
}

func insertOrganization(ctx context.Context, db bun.IDB, org *data.Organization) error {
	_, err := db.NewInsert().Model(org).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert organization: %w", err)
	}
	return nil
}
func insertMember(ctx context.Context, db bun.IDB, member *data.OrganizationMember) error {
	_, err := db.NewInsert().Model(member).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert member: %w", err)
	}
	return nil
}
