package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
)

const InvitationsTTL = 48 * time.Hour

var (
	ErrInvitationAlreadyProcessed = errors.New("invitation already processed")
	ErrInvitationNotFound         = errors.New("invitation not found")
	ErrInvitationAlreadyPending   = errors.New(
		"invitation already exists and is pending",
	)
	ErrInvitationExpired = errors.New("invitation expired")
	ErrNotInvitee        = errors.New("user is not invitee")
)

type InvitationDetails struct {
	Invitation       *domain.OrganizationInvitation
	OrganizationName string
}

func (s *OrganizationsService) CreateInvitation(
	ctx context.Context,
	organizationID uuid.UUID,
	inviterID uuid.UUID,
	inviteeEmail string,
) (*InvitationDetails, error) {
	if _, err := s.repo.GetMember(ctx, organizationID, inviterID); err != nil {
		return nil, ErrUserIsNotMember
	}

	exists, err := s.repo.HasPendingInvitation(
		ctx,
		organizationID,
		inviteeEmail,
	)
	if err != nil {
		return nil, fmt.Errorf("check pending invitations: %w", err)
	}

	if exists {
		return nil, ErrInvitationAlreadyPending
	}

	inv, err := s.repo.CreateInvitation(
		ctx,
		organizationID,
		inviteeEmail,
		time.Now().Add(InvitationsTTL),
	)
	if err != nil {
		return nil, fmt.Errorf("create invitation: %w", err)
	}

	return s.invitationDetails(ctx, inv), nil
}

func (s *OrganizationsService) ListMyInvitations(
	ctx context.Context,
	email string,
) ([]InvitationDetails, error) {
	invs, err := s.repo.ListInvitationsForUser(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("list user invitations: %w", err)
	}

	return s.invitationDetailsList(ctx, invs), nil
}

func (s *OrganizationsService) ListOrgInvitations(
	ctx context.Context,
	organizationID uuid.UUID,
	userID uuid.UUID,
) ([]InvitationDetails, error) {
	member, err := s.repo.GetMember(ctx, organizationID, userID)
	if err != nil {
		return nil, ErrUserIsNotMember
	}

	if member.Role != domain.RoleOwner &&
		member.Role != domain.RoleAdministrator {
		return nil, ErrNotAllowed
	}

	invs, err := s.repo.ListInvitationsForOrg(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list organization invitations: %w", err)
	}

	return s.invitationDetailsList(ctx, invs), nil
}

func (s *OrganizationsService) AcceptInvitation(
	ctx context.Context,
	invitationID uuid.UUID,
	userID uuid.UUID,
	userEmail string,
) error {
	inv, err := s.repo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return ErrInvitationNotFound
	}

	if inv.Status != domain.InvStatusPending {
		return ErrInvitationAlreadyProcessed
	}

	if time.Now().After(inv.ExpiresAt) {
		return ErrInvitationExpired
	}

	if userEmail != inv.InviteeEmail {
		return ErrNotInvitee
	}

	alreadyMember, err := s.repo.UserIsMember(ctx, userID, inv.OrganizationID)
	if err != nil {
		return fmt.Errorf("check member: %w", err)
	}

	if alreadyMember {
		return ErrMemberAlreadyExists
	}

	if err := s.repo.UpdateInvitationStatus(
		ctx,
		invitationID,
		domain.InvStatusAccepted,
	); err != nil {
		return fmt.Errorf("accept invitation: %w", err)
	}

	if err := s.repo.AddMember(ctx, &domain.OrganizationMember{
		MemberID:       userID,
		OrganizationID: inv.OrganizationID,
		Role:           domain.RoleMember,
		JoinedAt:       time.Now(),
	}); err != nil {
		return fmt.Errorf("add invited member: %w", err)
	}

	return nil
}

func (s *OrganizationsService) DeclineInvitation(
	ctx context.Context,
	invitationID uuid.UUID,
	userEmail string,
) error {
	inv, err := s.repo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return ErrInvitationNotFound
	}

	if inv.Status != domain.InvStatusPending {
		return ErrInvitationAlreadyProcessed
	}

	if userEmail != inv.InviteeEmail {
		return ErrNotInvitee
	}

	if err := s.repo.UpdateInvitationStatus(
		ctx,
		invitationID,
		domain.InvStatusDeclined,
	); err != nil {
		return fmt.Errorf("decline invitation: %w", err)
	}

	return nil
}

func (s *OrganizationsService) invitationDetailsList(
	ctx context.Context,
	invs []domain.OrganizationInvitation,
) []InvitationDetails {
	details := make([]InvitationDetails, 0, len(invs))
	for i := range invs {
		details = append(details, *s.invitationDetails(ctx, &invs[i]))
	}

	return details
}

func (s *OrganizationsService) invitationDetails(
	ctx context.Context,
	inv *domain.OrganizationInvitation,
) *InvitationDetails {
	orgName := ""
	if org, err := s.repo.GetOrganizationByID(
		ctx,
		inv.OrganizationID,
	); err == nil {
		orgName = org.Name
	}

	return &InvitationDetails{
		Invitation:       inv,
		OrganizationName: orgName,
	}
}
