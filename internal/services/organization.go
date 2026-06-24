package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/paging"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
)

const PageSize = 100

var ErrUserIsNotMember = errors.New("user is not member of organization")

type OrganizationsRepository interface {
	CreateOrganization(ctx context.Context, name string, creatorID uuid.UUID) (uuid.UUID, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*data.Organization, error)
	UserIsMember(ctx context.Context, userID, orgID uuid.UUID) (bool, error)
	GetUsersOrganizations(
		ctx context.Context,
		userID uuid.UUID,
		p paging.Paging,
	) ([]*data.Organization, error)
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
	organizationID uuid.UUID,
	userID uuid.UUID,
) (*data.Organization, error) {
	if isMember, err := s.repo.UserIsMember(ctx, userID, organizationID); err != nil {
		return nil, fmt.Errorf("failed check user membership of %q: %w", organizationID, err)
	} else if !isMember {
		return nil, ErrUserIsNotMember
	}

	org, err := s.repo.GetOrganizationByID(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
	}

	return org, nil
}

func (s *OrganizationsService) GetUsersOrganizations(
	ctx context.Context,
	userID uuid.UUID,
	page int,
) ([]*data.Organization, error) {
	orgs, err := s.repo.GetUsersOrganizations(
		ctx,
		userID,
		paging.Paging{Page: page, Size: PageSize},
	)
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
