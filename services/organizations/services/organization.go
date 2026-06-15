package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
)

type OrganizationsRepository interface {
	CreateOrganization(ctx context.Context, name string, creatorID uuid.UUID) (uuid.UUID, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*data.Organization, error)
	UserIsMember(ctx context.Context, userID, orgID uuid.UUID) (bool, error)
	GetUsersOrganizations(ctx context.Context, userID uuid.UUID) ([]data.Organization, error)
}

type OrganizationsService struct {
	repo OrganizationsRepository
}

func NewOrganizations(repo OrganizationsRepository) *OrganizationsService {
	return &OrganizationsService{
		repo: repo,
	}
}

// CreateOrganization implements [handlers.OrganizationService].
func (s *OrganizationsService) CreateOrganization(ctx context.Context,
	name string,
	creatorID uuid.UUID,
) (uuid.UUID, error) {
	id, err := s.repo.CreateOrganization(ctx, name, creatorID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed save organization: %w", err)
	}

	return id, nil
}

func (s *OrganizationsService) GetOrganization(
	ctx context.Context,
	userID uuid.UUID,
	id uuid.UUID,
) (*data.Organization, error) {
	if isMember, err := s.repo.UserIsMember(ctx, userID, id); err != nil {
		return nil, fmt.Errorf("failed check user membership of %q: %w", id, err)
	} else if !isMember {
		return nil, fmt.Errorf("user is not member of organization %q", id)
	}

	org, err := s.repo.GetOrganizationByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
	}

	return org, nil
}

func (s *OrganizationsService) GetUsersOrganizations(
	ctx context.Context,
	userID uuid.UUID,
	page int,
) ([]data.Organization, error) {
	orgs, err := s.repo.GetUsersOrganizations(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get users orgs: %w", err)
	}

	return orgs, nil
}

func (s *OrganizationsService) IsMember(
	ctx context.Context,
	userID, orgID uuid.UUID,
) (bool, error) {
	isMember, err := s.repo.UserIsMember(ctx, userID, orgID)
	if err != nil {
		return false, fmt.Errorf("failed check user membership of %q: %w", orgID, err)
	}

	return isMember, nil
}
