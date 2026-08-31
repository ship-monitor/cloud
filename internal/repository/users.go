package repository

import (
	"context"
	"database/sql"
	"fmt"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/domain"
	"github.com/ship-monitor/cloud/internal/repository/common"
	"github.com/uptrace/bun"
)

type UsersRepo struct {
	db *bun.DB
}

func NewUsers(db *sql.DB) *UsersRepo {
	return &UsersRepo{db: common.NewBunDB(db)}
}

func (u *UsersRepo) Migrate(ctx context.Context) error {
	_, err := u.db.NewCreateTable().
		Model((*domain.User)(nil)).
		IfNotExists().Exec(ctx)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	_, err = u.db.NewCreateIndex().
		IfNotExists().
		Model((*domain.User)(nil)).
		Index("idx_users_email").
		Column("email").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	return nil
}

func (u *UsersRepo) GetUser(
	ctx context.Context,
	userID uuid.UUID,
) (*domain.User, error) {
	var user domain.User

	err := u.db.NewSelect().
		Model(&user).
		Where("id = ?", userID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select user: %w", err)
	}

	return &user, nil
}

func (u *UsersRepo) GetUserByEmail(
	ctx context.Context,
	email string,
) (*domain.User, error) {
	var user domain.User

	err := u.db.NewSelect().
		Model(&user).
		Where("email = ?", domain.NormalizeEmail(email)).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select user: %w", err)
	}

	return &user, nil
}

func (u *UsersRepo) CreateUser(ctx context.Context, user *domain.User) error {
	_, err := u.db.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

func (u *UsersRepo) SetEmailVerified(
	ctx context.Context,
	userID uuid.UUID,
	verified bool,
) error {
	query := u.db.NewUpdate().
		Model(&domain.User{ID: userID}).
		WherePK().
		Set("email_verified = ?", verified)

	_, err := query.Exec(ctx)
	if err != nil {
		log.Info("SetEmailVerified", "query", query.String())

		return fmt.Errorf("update email verified: %w", err)
	}

	return nil
}

func (u *UsersRepo) SetPassword(
	ctx context.Context,
	userID uuid.UUID,
	hashed []byte,
) error {
	_, err := u.db.NewUpdate().
		Model(&domain.User{ID: userID}).
		WherePK().
		Set("password_hash = ?", hashed).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	return nil
}

func (u *UsersRepo) SetEmail(
	ctx context.Context,
	userID uuid.UUID,
	email string,
) error {
	_, err := u.db.NewUpdate().
		Model(&domain.User{ID: userID}).
		WherePK().
		Set("email = ?", domain.NormalizeEmail(email)).
		Set("email_verified = ?", false).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update email: %w", err)
	}

	return nil
}

func (u *UsersRepo) EmailTaken(
	ctx context.Context,
	email string,
) (bool, error) {
	taken, err := u.db.NewSelect().
		Model((*domain.User)(nil)).
		Column("email").
		Where("email = ?", domain.NormalizeEmail(email)).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("select user by email: %w", err)
	}

	return taken, nil
}
