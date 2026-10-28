// Package user provides a sqlx-based repository for the users table.
//
//nolint:varnamelen
package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/user"
)

// ErrNotFound is returned when no user matches the given constraints.
var ErrNotFound = errors.New("user not found")

// Repository reads and writes users in the database.
//
// It depends only on sqlx.ExtContext, so it works with both *sqlx.DB and
// *sqlx.Tx, letting callers run multiple operations in one transaction.
type Repository struct {
	db sqlx.ExtContext
}

// New returns a Repository backed by the given database.
func New(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// WithTx returns a Repository that runs all its queries within tx.
func (r *Repository) WithTx(tx *sqlx.Tx) *Repository {
	return &Repository{db: tx}
}

// row mirrors the columns of the users table.
type row struct {
	ID           uuid.UUID `db:"id"`
	Name         string    `db:"name"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
}

func (r row) toDomain() domain.User {
	return domain.User{
		ID:           r.ID,
		Name:         r.Name,
		Email:        r.Email,
		PasswordHash: r.PasswordHash,
	}
}

const columns = "id, name, email, password_hash"

// Create inserts a new user.
func (r *Repository) Create(ctx context.Context, u domain.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, name, email, password_hash)
		 VALUES (?, ?, ?, ?)`,
		u.ID, u.Name, u.Email, u.PasswordHash,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

// Get returns the user with the given id, or ErrNotFound.
func (r *Repository) Get(
	ctx context.Context,
	id uuid.UUID,
) (domain.User, error) {
	var rw row

	err := r.db.
		QueryRowxContext(ctx, `SELECT `+columns+` FROM users WHERE id = ?`, id).
		StructScan(&rw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}

	if err != nil {
		return domain.User{}, fmt.Errorf("select user: %w", err)
	}

	return rw.toDomain(), nil
}

// GetByEmail returns the user with the given email, or ErrNotFound.
func (r *Repository) GetByEmail(
	ctx context.Context,
	email string,
) (domain.User, error) {
	var rw row

	err := r.db.
		QueryRowxContext(
			ctx,
			`SELECT `+columns+` FROM users WHERE email = ?`,
			email,
		).
		StructScan(&rw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}

	if err != nil {
		return domain.User{}, fmt.Errorf("select user by email: %w", err)
	}

	return rw.toDomain(), nil
}

// List returns all users ordered by email.
func (r *Repository) List(ctx context.Context) ([]domain.User, error) {
	var rows []row

	err := sqlx.SelectContext(ctx, r.db, &rows,
		`SELECT `+columns+` FROM users ORDER BY email`,
	)
	if err != nil {
		return nil, fmt.Errorf("select users: %w", err)
	}

	users := make([]domain.User, len(rows))
	for i, rw := range rows {
		users[i] = rw.toDomain()
	}

	return users, nil
}

// EmailTaken reports whether a user with the given email exists.
func (r *Repository) EmailTaken(
	ctx context.Context,
	email string,
) (bool, error) {
	var taken bool

	err := r.db.
		QueryRowxContext(
			ctx,
			`SELECT EXISTS (SELECT 1 FROM users WHERE email = ?)`,
			email,
		).
		Scan(&taken)
	if err != nil {
		return false, fmt.Errorf("check email taken: %w", err)
	}

	return taken, nil
}

// Delete removes the user with the given id, or ErrNotFound.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}
