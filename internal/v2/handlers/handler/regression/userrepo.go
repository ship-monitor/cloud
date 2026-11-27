package regression

import (
	"context"

	"github.com/lib/pq"
	"github.com/ship-monitor/cloud/internal/v2/domain/user"
	userService "github.com/ship-monitor/cloud/internal/v2/services/user"
)

type UserRepository struct {
	userService.Repository

	checks  int
	creates int
}

func (r *UserRepository) Checks() int {
	return r.checks
}

func (r *UserRepository) Creates() int {
	return r.creates
}

func (r *UserRepository) EmailTaken(
	_ context.Context,
	_ string,
) (bool, error) {
	r.checks++

	return false, nil
}

func (r *UserRepository) Create(
	_ context.Context,
	_ user.User,
) error {
	r.creates++

	//nolint:exhaustruct_v5
	return &pq.Error{
		Code:       "23505",
		Constraint: "users_email_key",
		Severity:   pq.Efatal,
		Message:    "unique constraints failed",
	}
}
