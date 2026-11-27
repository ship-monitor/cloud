package regression

import (
	"context"

	"github.com/ship-monitor/cloud/internal/v2/domain/user"
	"github.com/ship-monitor/cloud/internal/v2/handlers/handler"
)

var _ handler.UserService = (*UserService)(nil)

type UserService struct {
	handler.UserService

	registerErr error
	loginErr    error
	calls       int
}

//nolint:exhaustruct_v5
func NewUserServiceWithRegisterErr(err error) *UserService {
	return &UserService{
		registerErr: err,
	}
}

//nolint:exhaustruct_v5
func NewUserServiceWithLoginErr(err error) *UserService {
	return &UserService{
		loginErr: err,
	}
}

func (s *UserService) Register(
	_ context.Context,
	_, _, _ string,
) (user.User, error) {
	s.calls++

	return user.User{}, s.registerErr
}

func (s *UserService) Login(
	_ context.Context,
	_, _ string,
) (user.User, error) {
	return user.User{}, s.loginErr
}

func (s *UserService) Calls() int {
	return s.calls
}
