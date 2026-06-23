package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/paging"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
)

type OrganizationID = uuid.UUID

type OrganizationsRepo struct {
	db *bun.DB
}

func New(db *sql.DB) *OrganizationsRepo {
	if db == nil {
		panic("db is nil")
	}

	return &OrganizationsRepo{
		db: bun.NewDB(db, sqlitedialect.New()),
	}
}

func (r *OrganizationsRepo) Migrate(ctx context.Context) error {
	models := []any{
		&data.Organization{},
		&data.OrganizationInvitation{},
		&data.OrganizationMember{},
		&data.OrganizationDevice{},
	}

	for _, model := range models {
		_, err := r.db.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("migrate model: %w", err)
		}
	}

	return nil
}

func (r *OrganizationsRepo) CreateOrganization(
	ctx context.Context,
	name string,
	creatorID uuid.UUID,
) (OrganizationID, error) {
	orgID := uuid.New()

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		err := insertOrganization(ctx, tx, &data.Organization{
			ID:        orgID,
			Name:      name,
			CreatorID: creatorID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
		if err != nil {
			return err
		}

		err = insertMember(ctx, tx, &data.OrganizationMember{
			MemberID:       creatorID,
			OrganizationID: orgID,
			JoinedAt:       time.Now(),
			Role:           data.RoleOwner,
		})
		if err != nil {
			return err
		}

		return nil
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

func (r *OrganizationsRepo) GetOrganizationByID(
	ctx context.Context,
	id OrganizationID,
) (*data.Organization, error) {
	org := data.Organization{}

	err := r.db.NewSelect().Model(&org).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get org by id: %w", err)
	}

	return &org, nil
}

func (r *OrganizationsRepo) UserIsMember(
	ctx context.Context,
	userID uuid.UUID,
	orgID OrganizationID,
) (bool, error) {
	exists, err := r.db.NewSelect().
		Model(&data.OrganizationMember{}).
		Where("member_id = ?", userID).
		Where("organization_id = ?", orgID).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check member record exist: %w", err)
	}

	return exists, nil
}

// GetUsersOrganizations implements [organization.Repository].
func (r *OrganizationsRepo) GetUsersOrganizations(
	ctx context.Context,
	userID uuid.UUID, p paging.Paging,
) ([]*data.Organization, error) {
	var orgs []*data.Organization

	err := r.db.NewSelect().
		Model(&orgs).
		Relation("OrganizationMembers", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("member_id = ?", userID)
		}).
		Limit(p.Size).
		Offset(p.Page * p.Size).
		Distinct().Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select users organization: %w", err)
	}

	return orgs, nil
}
