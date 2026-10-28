// Package user contains the user business logic.
package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/user"
	userrepo "github.com/ship-monitor/cloud/internal/v2/repositories/user"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrEmailTaken is returned when a user with the given email already
	// exists.
	ErrEmailTaken = errors.New("email is already taken")

	// ErrInvalidCredentials is returned when the email or password is wrong.
	// Deliberately vague, so it does not reveal which of the two it was.
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// Repository is the persistence interface of the Service.
type Repository interface {
	Create(ctx context.Context, u domain.User) error
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	EmailTaken(ctx context.Context, email string) (bool, error)
}

// Service implements user use cases.
type Service struct {
	repo Repository
}

// New returns a Service backed by the given repository.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// Register creates a new user with a bcrypt-hashed password.
func (s *Service) Register(
	ctx context.Context,
	email, password, name string,
) (domain.User, error) {
	if email == "" || password == "" || name == "" {
		return domain.User{}, errors.New(
			"email, password and name are required",
		)
	}

	taken, err := s.repo.EmailTaken(ctx, email)
	if err != nil {
		return domain.User{}, fmt.Errorf("check email taken: %w", err)
	}

	if taken {
		return domain.User{}, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash password: %w", err)
	}

	u := domain.User{
		ID:           uuid.New(),
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	return u, nil
}

// Login returns the user with the given email if the password matches.
func (s *Service) Login(
	ctx context.Context,
	email, password string,
) (domain.User, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if errors.Is(err, userrepo.ErrNotFound) {
		return domain.User{}, ErrInvalidCredentials
	}

	if err != nil {
		return domain.User{}, fmt.Errorf("get user by email: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(u.PasswordHash),
		[]byte(password),
	); err != nil {
		return domain.User{}, ErrInvalidCredentials
	}

	return u, nil
}
