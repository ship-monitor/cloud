package user

import (
	"context"
	"errors"
	"os/user"

	"github.com/google/uuid"
)

var errNotImplemented = errors.New("not implemented")

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) GetUser(
	ctx context.Context,
	id uuid.UUID,
) (*user.User, error) {
	return nil, errNotImplemented
}

func (s *Service) GetUserByEmail(
	ctx context.Context,
	eail string,
) (*user.User, error) {
	return nil, errNotImplemented
}
