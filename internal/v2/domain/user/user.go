package user

import (
	"errors"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string
}

var (
	// ErrNotFound is returned when no user matches the given constraints.
	ErrNotFound = errors.New("user not found")

	// ErrEmailTaken is returned when a user with the given email already
	// exists.
	ErrEmailTaken = errors.New("email already taken")

	// ErrInvalidCredentials is returned when the email or password is wrong.
	// Deliberately vague, so it does not reveal which of the two it was.
	ErrInvalidCredentials = errors.New("invalid email or password")
)
