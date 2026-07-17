package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/paging"
)

const PageSize = 100

var (
	ErrUserIsNotMember = errors.New(
		"user is not member of organization",
	)
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrOnlyOwnerCanDelete   = errors.New(
		"only owner can delete organization",
	)
	ErrNotAllowed            = errors.New("not allowed")
	ErrMemberNotFound        = errors.New("member not found")
	ErrMemberAlreadyExists   = errors.New("user is already a member")
	ErrCannotAssignOwnerRole = errors.New("cannot assign owner role")
	ErrCannotChangeOwnerRole = errors.New("cannot change owner role")
	ErrInvalidMemberRole     = errors.New(
		"invalid role, allowed: administrator, member",
	)
	ErrRemovingOrganizationOwner = errors.New("removing owner of organization")
)

type FullOrganizationsRepository interface {
	OrganizationRepository
	OrganizationMembersRepository
	OrganizationInvitationsRepository
}

type OrganizationRepository interface {
	CreateOrganization(
		ctx context.Context,
		name string,
		creatorID uuid.UUID,
	) (uuid.UUID, error)
	GetOrganizationByID(
		ctx context.Context,
		id uuid.UUID,
	) (*domain.Organization, error)
	UserIsMember(ctx context.Context, userID, orgID uuid.UUID) (bool, error)
	GetUsersOrganizations(
		ctx context.Context,
		userID uuid.UUID,
		p paging.Paging,
	) ([]*domain.Organization, error)
	RenameOrganization(ctx context.Context, orgID uuid.UUID, name string) error
	DeleteOrganization(ctx context.Context, orgID uuid.UUID) error
}

type OrganizationMembersRepository interface {
	AddMember(ctx context.Context, member *domain.OrganizationMember) error
	UpdateMemberRole(
		ctx context.Context,
		orgID, userID uuid.UUID,
		role domain.Role,
	) error
	RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error
	GetMember(
		ctx context.Context,
		orgID, userID uuid.UUID,
	) (*domain.OrganizationMember, error)
	GetMembersWithUserInfo(
		ctx context.Context,
		orgID uuid.UUID,
	) ([]domain.OrganizationMemberWithUser, error)
}

type OrganizationInvitationsRepository interface {
	CreateInvitation(
		ctx context.Context,
		orgID uuid.UUID,
		inviteeEmail string,
		expiresAt time.Time,
	) (*domain.OrganizationInvitation, error)
	GetInvitationByID(
		ctx context.Context,
		id uuid.UUID,
	) (*domain.OrganizationInvitation, error)
	HasPendingInvitation(
		ctx context.Context,
		orgID uuid.UUID,
		email string,
	) (bool, error)
	ListInvitationsForUser(
		ctx context.Context,
		email string,
	) ([]domain.OrganizationInvitation, error)
	ListInvitationsForOrg(
		ctx context.Context,
		orgID uuid.UUID,
	) ([]domain.OrganizationInvitation, error)
	UpdateInvitationStatus(
		ctx context.Context,
		id uuid.UUID,
		status domain.InvitationStatus,
	) error
}

type OrganizationsService struct {
	repo FullOrganizationsRepository
}

func NewOrganizations(repo FullOrganizationsRepository) *OrganizationsService {
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
) (*domain.Organization, error) {
	if isMember, err := s.repo.UserIsMember(
		ctx,
		userID,
		organizationID,
	); err != nil {
		return nil, fmt.Errorf(
			"failed check user membership of %q: %w",
			organizationID,
			err,
		)
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
) ([]*domain.Organization, error) {
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
		return false, fmt.Errorf(
			"failed check user membership of %q: %w",
			orgID,
			err,
		)
	}

	return isMember, nil
}

func (s *OrganizationsService) RenameOrganization(
	ctx context.Context,
	organizationID uuid.UUID,
	name string,
	userID uuid.UUID,
) error {
	if isMember, err := s.repo.UserIsMember(
		ctx,
		userID,
		organizationID,
	); err != nil {
		return fmt.Errorf(
			"failed check user membership of %q: %w",
			organizationID,
			err,
		)
	} else if !isMember {
		return ErrUserIsNotMember
	}

	err := s.repo.RenameOrganization(ctx, organizationID, name)
	if err != nil {
		return fmt.Errorf("rename organization: %w", err)
	}

	return nil
}

func (s *OrganizationsService) DeleteOrganization(
	ctx context.Context,
	organizationID uuid.UUID,
	userID uuid.UUID,
) error {
	org, err := s.repo.GetOrganizationByID(ctx, organizationID)
	if err != nil {
		return ErrOrganizationNotFound
	}

	if org.CreatorID != userID {
		return ErrOnlyOwnerCanDelete
	}

	if err := s.repo.DeleteOrganization(ctx, organizationID); err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}

	return nil
}

func (s *OrganizationsService) GetMembers(
	ctx context.Context,
	organizationID uuid.UUID,
	userID uuid.UUID,
) ([]domain.OrganizationMemberWithUser, error) {
	if isMember, err := s.repo.UserIsMember(
		ctx,
		userID,
		organizationID,
	); err != nil {
		return nil, fmt.Errorf(
			"failed check user membership of %q: %w",
			organizationID,
			err,
		)
	} else if !isMember {
		return nil, ErrUserIsNotMember
	}

	members, err := s.repo.GetMembersWithUserInfo(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}

	return members, nil
}

func (s *OrganizationsService) AddMember(
	ctx context.Context,
	organizationID uuid.UUID,
	actorID uuid.UUID,
	userID uuid.UUID,
	role domain.Role,
) (*domain.OrganizationMember, error) {
	if _, err := s.repo.GetMember(ctx, organizationID, actorID); err != nil {
		return nil, ErrUserIsNotMember
	}

	if err := validateAssignableMemberRole(role); err != nil {
		return nil, err
	}

	isMember, err := s.repo.UserIsMember(ctx, userID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("check member: %w", err)
	}

	if isMember {
		return nil, ErrMemberAlreadyExists
	}

	member := domain.OrganizationMember{
		MemberID:       userID,
		OrganizationID: organizationID,
		Role:           role,
		JoinedAt:       time.Now(),
	}

	if err := s.repo.AddMember(ctx, &member); err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	return &member, nil
}

func (s *OrganizationsService) UpdateMemberRole(
	ctx context.Context,
	organizationID uuid.UUID,
	actorID uuid.UUID,
	userID uuid.UUID,
	role domain.Role,
) error {
	if _, err := s.repo.GetMember(ctx, organizationID, actorID); err != nil {
		return ErrUserIsNotMember
	}

	targetMember, err := s.repo.GetMember(ctx, organizationID, userID)
	if err != nil {
		return ErrMemberNotFound
	}

	if targetMember.Role == domain.RoleOwner {
		return ErrCannotChangeOwnerRole
	}

	if err := validateAssignableMemberRole(role); err != nil {
		return err
	}

	if err := s.repo.UpdateMemberRole(
		ctx,
		organizationID,
		userID,
		role,
	); err != nil {
		return fmt.Errorf("update member role: %w", err)
	}

	return nil
}

func (s *OrganizationsService) RemoveMember(
	ctx context.Context,
	organizationID uuid.UUID,
	actorID uuid.UUID,
	userID uuid.UUID,
) error {
	currentMember, err := s.repo.GetMember(ctx, organizationID, actorID)
	if err != nil {
		return ErrUserIsNotMember
	}

	if currentMember.Role != domain.RoleOwner &&
		currentMember.Role != domain.RoleAdministrator {
		return fmt.Errorf(
			"only owner or administrator can remove members: %w",
			ErrNotAllowed,
		)
	}

	targetMember, err := s.repo.GetMember(ctx, organizationID, userID)
	if err != nil {
		return ErrMemberNotFound
	}

	if targetMember.Role == domain.RoleOwner {
		return ErrRemovingOrganizationOwner
	}

	if err := s.repo.RemoveMember(ctx, organizationID, userID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	return nil
}

func validateAssignableMemberRole(role domain.Role) error {
	switch role {
	case domain.RoleOwner:
		return ErrCannotAssignOwnerRole
	case domain.RoleAdministrator, domain.RoleMember:
		return nil
	default:
		return ErrInvalidMemberRole
	}
}
