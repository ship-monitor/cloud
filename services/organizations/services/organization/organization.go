package organization

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/handlers"
)

type Repository interface {
	CreateOrganization(ctx context.Context, name string, creatorID uuid.UUID) (uuid.UUID, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// CreateOrganization implements [handlers.OrganizationService].
func (s *Service) CreateOrganization(name string, creatorID uuid.UUID) (handlers.OrganizationID, error) {

	id, err := s.repo.CreateOrganization(context.TODO(), name, creatorID)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed save organization: %w", err)
	}
	return id, nil
}
