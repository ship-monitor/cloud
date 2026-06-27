package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/paging"
)

func TestOrganizationsServiceOrganizationWorkflow(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	repo := newFakeOrganizationsRepository()
	orgID := uuid.New()
	ownerID := uuid.New()
	memberID := uuid.New()
	outsiderID := uuid.New()

	repo.addOrg(orgID, "Acme", ownerID)
	repo.addMember(orgID, ownerID, domain.RoleOwner)
	repo.addMember(orgID, memberID, domain.RoleMember)

	svc := services.NewOrganizations(repo)

	org, err := svc.GetOrganization(ctx, orgID, memberID)
	if err != nil {
		t.Fatalf("get organization as member: %v", err)
	}

	if org.Name != "Acme" {
		t.Fatalf("expected organization name Acme, got %q", org.Name)
	}

	_, err = svc.GetOrganization(ctx, orgID, outsiderID)
	if !errors.Is(err, services.ErrUserIsNotMember) {
		t.Fatalf("expected ErrUserIsNotMember, got %v", err)
	}

	if err := svc.RenameOrganization(ctx, orgID, "Renamed", memberID); err != nil {
		t.Fatalf("rename organization: %v", err)
	}

	if repo.orgs[orgID].Name != "Renamed" {
		t.Fatalf("expected organization to be renamed, got %q", repo.orgs[orgID].Name)
	}

	err = svc.DeleteOrganization(ctx, orgID, memberID)
	if !errors.Is(err, services.ErrOnlyOwnerCanDelete) {
		t.Fatalf("expected ErrOnlyOwnerCanDelete, got %v", err)
	}

	if err := svc.DeleteOrganization(ctx, orgID, ownerID); err != nil {
		t.Fatalf("delete organization as owner: %v", err)
	}

	if _, ok := repo.orgs[orgID]; ok {
		t.Fatal("expected organization to be deleted")
	}

	if _, ok := repo.members[memberKey{orgID: orgID, userID: ownerID}]; ok {
		t.Fatal("expected organization owner membership to be deleted")
	}

	err = svc.DeleteOrganization(ctx, uuid.New(), ownerID)
	if !errors.Is(err, services.ErrOrganizationNotFound) {
		t.Fatalf("expected ErrOrganizationNotFound, got %v", err)
	}
}

func TestOrganizationsServiceMemberWorkflow(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	repo := newFakeOrganizationsRepository()
	orgID := uuid.New()
	ownerID := uuid.New()
	adminID := uuid.New()
	memberID := uuid.New()
	newMemberID := uuid.New()
	plainMemberID := uuid.New()
	outsiderID := uuid.New()

	repo.addOrg(orgID, "Acme", ownerID)
	repo.addMember(orgID, ownerID, domain.RoleOwner)
	repo.addMember(orgID, adminID, domain.RoleAdministrator)
	repo.addMember(orgID, memberID, domain.RoleMember)
	repo.addMember(orgID, plainMemberID, domain.RoleMember)

	svc := services.NewOrganizations(repo)

	created, err := svc.AddMember(ctx, orgID, ownerID, newMemberID, domain.RoleAdministrator)
	if err != nil {
		t.Fatalf("add member: %v", err)
	}

	if created.MemberID != newMemberID || created.Role != domain.RoleAdministrator {
		t.Fatalf("unexpected created member: %+v", created)
	}

	_, err = svc.AddMember(ctx, orgID, ownerID, newMemberID, domain.RoleMember)
	if !errors.Is(err, services.ErrMemberAlreadyExists) {
		t.Fatalf("expected ErrMemberAlreadyExists, got %v", err)
	}

	_, err = svc.AddMember(ctx, orgID, ownerID, uuid.New(), domain.RoleOwner)
	if !errors.Is(err, services.ErrCannotAssignOwnerRole) {
		t.Fatalf("expected ErrCannotAssignOwnerRole, got %v", err)
	}

	_, err = svc.AddMember(ctx, orgID, outsiderID, uuid.New(), domain.RoleMember)
	if !errors.Is(err, services.ErrUserIsNotMember) {
		t.Fatalf("expected ErrUserIsNotMember, got %v", err)
	}

	if err := svc.UpdateMemberRole(
		ctx,
		orgID,
		ownerID,
		memberID,
		domain.RoleAdministrator,
	); err != nil {
		t.Fatalf("update member role: %v", err)
	}

	if repo.members[memberKey{orgID: orgID, userID: memberID}].Role != domain.RoleAdministrator {
		t.Fatal("expected member role to be updated")
	}

	err = svc.UpdateMemberRole(ctx, orgID, adminID, ownerID, domain.RoleMember)
	if !errors.Is(err, services.ErrCannotChangeOwnerRole) {
		t.Fatalf("expected ErrCannotChangeOwnerRole, got %v", err)
	}

	err = svc.UpdateMemberRole(ctx, orgID, ownerID, memberID, domain.Role("captain"))
	if !errors.Is(err, services.ErrInvalidMemberRole) {
		t.Fatalf("expected ErrInvalidMemberRole, got %v", err)
	}

	err = svc.RemoveMember(ctx, orgID, plainMemberID, memberID)
	if !errors.Is(err, services.ErrNotAllowed) {
		t.Fatalf("expected ErrNotAllowed, got %v", err)
	}

	err = svc.RemoveMember(ctx, orgID, adminID, ownerID)
	if !errors.Is(err, services.ErrRemovingOrganizationOwner) {
		t.Fatalf("expected ErrRemovingOrganizationOwner, got %v", err)
	}

	if err := svc.RemoveMember(ctx, orgID, ownerID, memberID); err != nil {
		t.Fatalf("remove member: %v", err)
	}

	if _, ok := repo.members[memberKey{orgID: orgID, userID: memberID}]; ok {
		t.Fatal("expected member to be removed")
	}

	err = svc.RemoveMember(ctx, orgID, ownerID, uuid.New())
	if !errors.Is(err, services.ErrMemberNotFound) {
		t.Fatalf("expected ErrMemberNotFound, got %v", err)
	}
}

func TestOrganizationsServiceInvitationWorkflow(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	repo := newFakeOrganizationsRepository()
	orgID := uuid.New()
	ownerID := uuid.New()
	memberID := uuid.New()
	inviteeID := uuid.New()
	inviteeEmail := "invitee@example.com"

	repo.addOrg(orgID, "Acme", ownerID)
	repo.addMember(orgID, ownerID, domain.RoleOwner)
	repo.addMember(orgID, memberID, domain.RoleMember)

	svc := services.NewOrganizations(repo)

	inv, err := svc.CreateInvitation(ctx, orgID, ownerID, inviteeEmail)
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}

	if inv.OrganizationName != "Acme" {
		t.Fatalf("expected organization name in invitation details, got %q", inv.OrganizationName)
	}

	if inv.Invitation.InviteeEmail != inviteeEmail ||
		inv.Invitation.Status != domain.InvStatusPending {
		t.Fatalf("unexpected invitation details: %+v", inv.Invitation)
	}

	_, err = svc.CreateInvitation(ctx, orgID, ownerID, inviteeEmail)
	if !errors.Is(err, services.ErrInvitationAlreadyPending) {
		t.Fatalf("expected ErrInvitationAlreadyPending, got %v", err)
	}

	_, err = svc.CreateInvitation(ctx, orgID, uuid.New(), "outsider@example.com")
	if !errors.Is(err, services.ErrUserIsNotMember) {
		t.Fatalf("expected ErrUserIsNotMember, got %v", err)
	}

	mine, err := svc.ListMyInvitations(ctx, inviteeEmail)
	if err != nil {
		t.Fatalf("list my invitations: %v", err)
	}

	if len(mine) != 1 || mine[0].Invitation.ID != inv.Invitation.ID {
		t.Fatalf("expected created invitation in user list, got %+v", mine)
	}

	orgInvs, err := svc.ListOrgInvitations(ctx, orgID, ownerID)
	if err != nil {
		t.Fatalf("list organization invitations: %v", err)
	}

	if len(orgInvs) != 1 || orgInvs[0].Invitation.ID != inv.Invitation.ID {
		t.Fatalf("expected created invitation in organization list, got %+v", orgInvs)
	}

	_, err = svc.ListOrgInvitations(ctx, orgID, memberID)
	if !errors.Is(err, services.ErrNotAllowed) {
		t.Fatalf("expected ErrNotAllowed, got %v", err)
	}

	err = svc.AcceptInvitation(ctx, inv.Invitation.ID, inviteeID, inviteeEmail)
	if err != nil {
		t.Fatalf("accept invitation: %v", err)
	}

	if repo.invitations[inv.Invitation.ID].Status != domain.InvStatusAccepted {
		t.Fatalf("expected accepted invitation, got %q", repo.invitations[inv.Invitation.ID].Status)
	}

	if repo.members[memberKey{orgID: orgID, userID: inviteeID}].Role != domain.RoleMember {
		t.Fatal("expected accepted invitee to be added as member")
	}

	err = svc.AcceptInvitation(ctx, inv.Invitation.ID, inviteeID, inviteeEmail)
	if !errors.Is(err, services.ErrInvitationAlreadyProcessed) {
		t.Fatalf("expected ErrInvitationAlreadyProcessed, got %v", err)
	}

	wrongInvitee := repo.addInvitation(
		orgID,
		"wrong@example.com",
		time.Now().Add(time.Hour),
	)

	err = svc.AcceptInvitation(ctx, wrongInvitee, uuid.New(), "other@example.com")
	if !errors.Is(err, services.ErrNotInvitee) {
		t.Fatalf("expected ErrNotInvitee, got %v", err)
	}

	expired := repo.addInvitation(
		orgID,
		"expired@example.com",
		time.Now().Add(-time.Hour),
	)

	err = svc.AcceptInvitation(ctx, expired, uuid.New(), "expired@example.com")
	if !errors.Is(err, services.ErrInvitationExpired) {
		t.Fatalf("expected ErrInvitationExpired, got %v", err)
	}

	alreadyMember := repo.addInvitation(
		orgID,
		"owner@example.com",
		time.Now().Add(time.Hour),
	)

	err = svc.AcceptInvitation(ctx, alreadyMember, ownerID, "owner@example.com")
	if !errors.Is(err, services.ErrMemberAlreadyExists) {
		t.Fatalf("expected ErrMemberAlreadyExists, got %v", err)
	}

	declined := repo.addInvitation(
		orgID,
		"decline@example.com",
		time.Now().Add(time.Hour),
	)
	if err := svc.DeclineInvitation(ctx, declined, "decline@example.com"); err != nil {
		t.Fatalf("decline invitation: %v", err)
	}

	if repo.invitations[declined].Status != domain.InvStatusDeclined {
		t.Fatalf("expected declined invitation, got %q", repo.invitations[declined].Status)
	}

	err = svc.DeclineInvitation(ctx, uuid.New(), "missing@example.com")
	if !errors.Is(err, services.ErrInvitationNotFound) {
		t.Fatalf("expected ErrInvitationNotFound, got %v", err)
	}
}

func TestOrgDevicesServiceWorkflow(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	orgRepo := newFakeOrganizationsRepository()
	devRepo := newFakeOrgDevicesRepository()
	orgID := uuid.New()
	memberID := uuid.New()
	outsiderID := uuid.New()
	deviceID := uuid.New()

	orgRepo.addOrg(orgID, "Acme", memberID)
	orgRepo.addMember(orgID, memberID, domain.RoleOwner)

	orgs := services.NewOrganizations(orgRepo)
	devices := services.NewOrgDevices(devRepo, orgs)

	if err := devices.ConnectDevice(ctx, deviceID, orgID, memberID, "Bridge"); err != nil {
		t.Fatalf("connect device: %v", err)
	}

	device := devRepo.devices[deviceID]
	if device.Name != "Bridge" || device.OrganizationID != orgID || device.CreatedAt.IsZero() {
		t.Fatalf("unexpected connected device: %+v", device)
	}

	err := devices.ConnectDevice(ctx, deviceID, orgID, memberID, "Duplicate")
	if !errors.Is(err, services.ErrAlreadyConnected) {
		t.Fatalf("expected ErrAlreadyConnected, got %v", err)
	}

	err = devices.ConnectDevice(ctx, uuid.New(), orgID, outsiderID, "No Access")
	if !errors.Is(err, services.ErrNotMember) {
		t.Fatalf("expected ErrNotMember, got %v", err)
	}

	listed, err := devices.GetDevices(ctx, orgID, memberID)
	if err != nil {
		t.Fatalf("get devices: %v", err)
	}

	if len(listed) != 1 || listed[0].ID != deviceID {
		t.Fatalf("expected connected device in list, got %+v", listed)
	}

	err = devices.RenameDevice(ctx, deviceID, memberID, "")
	if !errors.Is(err, services.ErrEmptyDeviceName) {
		t.Fatalf("expected ErrEmptyDeviceName, got %v", err)
	}

	if err := devices.RenameDevice(ctx, deviceID, memberID, "Engine Room"); err != nil {
		t.Fatalf("rename device: %v", err)
	}

	if devRepo.devices[deviceID].Name != "Engine Room" {
		t.Fatalf("expected renamed device, got %q", devRepo.devices[deviceID].Name)
	}

	_, err = devices.GetDevice(ctx, deviceID, outsiderID)
	if !errors.Is(err, services.ErrNotMember) {
		t.Fatalf("expected ErrNotMember, got %v", err)
	}

	if err := devices.DisconnectDevice(ctx, deviceID, memberID); err != nil {
		t.Fatalf("disconnect device: %v", err)
	}

	if _, ok := devRepo.devices[deviceID]; ok {
		t.Fatal("expected device to be deleted")
	}
}

type memberKey struct {
	orgID  uuid.UUID
	userID uuid.UUID
}

type fakeOrganizationsRepository struct {
	orgs        map[uuid.UUID]*domain.Organization
	members     map[memberKey]*domain.OrganizationMember
	invitations map[uuid.UUID]*domain.OrganizationInvitation
}

func newFakeOrganizationsRepository() *fakeOrganizationsRepository {
	return &fakeOrganizationsRepository{
		orgs:        make(map[uuid.UUID]*domain.Organization),
		members:     make(map[memberKey]*domain.OrganizationMember),
		invitations: make(map[uuid.UUID]*domain.OrganizationInvitation),
	}
}

func (r *fakeOrganizationsRepository) CreateOrganization(
	ctx context.Context,
	name string,
	creatorID uuid.UUID,
) (uuid.UUID, error) {
	_ = ctx
	id := uuid.New()
	r.addOrg(id, name, creatorID)
	r.addMember(id, creatorID, domain.RoleOwner)

	return id, nil
}

func (r *fakeOrganizationsRepository) GetOrganizationByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Organization, error) {
	_ = ctx

	org, ok := r.orgs[id]
	if !ok {
		return nil, errors.New("organization not found")
	}

	orgCopy := *org

	return &orgCopy, nil
}

func (r *fakeOrganizationsRepository) UserIsMember(
	ctx context.Context,
	userID, orgID uuid.UUID,
) (bool, error) {
	_ = ctx
	_, ok := r.members[memberKey{orgID: orgID, userID: userID}]

	return ok, nil
}

func (r *fakeOrganizationsRepository) GetUsersOrganizations(
	ctx context.Context,
	userID uuid.UUID,
	p paging.Paging,
) ([]*domain.Organization, error) {
	_ = ctx
	orgs := make([]*domain.Organization, 0)

	for key := range r.members {
		if key.userID != userID {
			continue
		}

		org, ok := r.orgs[key.orgID]
		if !ok {
			continue
		}

		orgCopy := *org
		orgs = append(orgs, &orgCopy)
	}

	start := p.Page * p.Size
	if start >= len(orgs) {
		return []*domain.Organization{}, nil
	}

	end := min(start+p.Size, len(orgs))

	return orgs[start:end], nil
}

func (r *fakeOrganizationsRepository) RenameOrganization(
	ctx context.Context,
	orgID uuid.UUID,
	name string,
) error {
	_ = ctx

	org, ok := r.orgs[orgID]
	if !ok {
		return errors.New("organization not found")
	}

	org.Name = name
	org.UpdatedAt = time.Now()

	return nil
}

func (r *fakeOrganizationsRepository) DeleteOrganization(
	ctx context.Context,
	orgID uuid.UUID,
) error {
	_ = ctx

	delete(r.orgs, orgID)

	for key := range r.members {
		if key.orgID == orgID {
			delete(r.members, key)
		}
	}

	return nil
}

func (r *fakeOrganizationsRepository) AddMember(
	ctx context.Context,
	member *domain.OrganizationMember,
) error {
	_ = ctx
	memberCopy := *member
	r.members[memberKey{orgID: member.OrganizationID, userID: member.MemberID}] = &memberCopy

	return nil
}

func (r *fakeOrganizationsRepository) UpdateMemberRole(
	ctx context.Context,
	orgID, userID uuid.UUID,
	role domain.Role,
) error {
	_ = ctx

	member, ok := r.members[memberKey{orgID: orgID, userID: userID}]
	if !ok {
		return errors.New("member not found")
	}

	member.Role = role

	return nil
}

func (r *fakeOrganizationsRepository) RemoveMember(
	ctx context.Context,
	orgID, userID uuid.UUID,
) error {
	_ = ctx

	delete(r.members, memberKey{orgID: orgID, userID: userID})

	return nil
}

func (r *fakeOrganizationsRepository) GetMember(
	ctx context.Context,
	orgID, userID uuid.UUID,
) (*domain.OrganizationMember, error) {
	_ = ctx

	member, ok := r.members[memberKey{orgID: orgID, userID: userID}]
	if !ok {
		return nil, errors.New("member not found")
	}

	memberCopy := *member

	return &memberCopy, nil
}

func (r *fakeOrganizationsRepository) GetMembersWithUserInfo(
	ctx context.Context,
	orgID uuid.UUID,
) ([]domain.OrganizationMemberWithUser, error) {
	_ = ctx

	members := make([]domain.OrganizationMemberWithUser, 0)
	for key, member := range r.members {
		if key.orgID != orgID {
			continue
		}

		members = append(members, domain.OrganizationMemberWithUser{
			MemberID:       member.MemberID,
			OrganizationID: member.OrganizationID,
			Role:           member.Role,
			JoinedAt:       member.JoinedAt,
			Name:           member.MemberID.String(),
			Email:          member.MemberID.String() + "@example.com",
		})
	}

	return members, nil
}

func (r *fakeOrganizationsRepository) CreateInvitation(
	ctx context.Context,
	orgID uuid.UUID,
	inviteeEmail string,
	expiresAt time.Time,
) (*domain.OrganizationInvitation, error) {
	_ = ctx
	id := r.addInvitation(orgID, inviteeEmail, expiresAt)
	invitationCopy := *r.invitations[id]

	return &invitationCopy, nil
}

func (r *fakeOrganizationsRepository) GetInvitationByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.OrganizationInvitation, error) {
	_ = ctx

	inv, ok := r.invitations[id]
	if !ok {
		return nil, errors.New("invitation not found")
	}

	invCopy := *inv

	return &invCopy, nil
}

func (r *fakeOrganizationsRepository) HasPendingInvitation(
	ctx context.Context,
	orgID uuid.UUID,
	email string,
) (bool, error) {
	_ = ctx

	for _, inv := range r.invitations {
		if inv.OrganizationID == orgID && inv.InviteeEmail == email &&
			inv.Status == domain.InvStatusPending {
			return true, nil
		}
	}

	return false, nil
}

func (r *fakeOrganizationsRepository) ListInvitationsForUser(
	ctx context.Context,
	email string,
) ([]domain.OrganizationInvitation, error) {
	_ = ctx

	invitations := make([]domain.OrganizationInvitation, 0)
	for _, inv := range r.invitations {
		if inv.InviteeEmail == email && inv.Status == domain.InvStatusPending {
			invitations = append(invitations, *inv)
		}
	}

	return invitations, nil
}

func (r *fakeOrganizationsRepository) ListInvitationsForOrg(
	ctx context.Context,
	orgID uuid.UUID,
) ([]domain.OrganizationInvitation, error) {
	_ = ctx

	invitations := make([]domain.OrganizationInvitation, 0)
	for _, inv := range r.invitations {
		if inv.OrganizationID == orgID {
			invitations = append(invitations, *inv)
		}
	}

	return invitations, nil
}

func (r *fakeOrganizationsRepository) UpdateInvitationStatus(
	ctx context.Context,
	id uuid.UUID,
	status domain.InvitationStatus,
) error {
	_ = ctx

	inv, ok := r.invitations[id]
	if !ok {
		return errors.New("invitation not found")
	}

	inv.Status = status

	return nil
}

func (r *fakeOrganizationsRepository) addOrg(id uuid.UUID, name string, creatorID uuid.UUID) {
	now := time.Now()
	r.orgs[id] = &domain.Organization{
		ID:        id,
		Name:      name,
		CreatorID: creatorID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (r *fakeOrganizationsRepository) addMember(
	orgID uuid.UUID,
	userID uuid.UUID,
	role domain.Role,
) {
	r.members[memberKey{orgID: orgID, userID: userID}] = &domain.OrganizationMember{
		MemberID:       userID,
		OrganizationID: orgID,
		Role:           role,
		JoinedAt:       time.Now(),
	}
}

func (r *fakeOrganizationsRepository) addInvitation(
	orgID uuid.UUID,
	email string,
	expiresAt time.Time,
) uuid.UUID {
	id := uuid.New()
	r.invitations[id] = &domain.OrganizationInvitation{
		ID:             id,
		OrganizationID: orgID,
		InviteeEmail:   email,
		Status:         domain.InvStatusPending,
		CreatedAt:      time.Now(),
		ExpiresAt:      expiresAt,
	}

	return id
}

type fakeOrgDevicesRepository struct {
	devices map[uuid.UUID]*domain.OrganizationDevice
}

func newFakeOrgDevicesRepository() *fakeOrgDevicesRepository {
	return &fakeOrgDevicesRepository{devices: make(map[uuid.UUID]*domain.OrganizationDevice)}
}

func (r *fakeOrgDevicesRepository) ListDevices(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]domain.OrganizationDevice, error) {
	_ = ctx

	devices := make([]domain.OrganizationDevice, 0)
	for _, device := range r.devices {
		if device.OrganizationID == organizationID {
			devices = append(devices, *device)
		}
	}

	return devices, nil
}

func (r *fakeOrgDevicesRepository) GetDevice(
	ctx context.Context,
	deviceID uuid.UUID,
) (*domain.OrganizationDevice, error) {
	_ = ctx

	device, ok := r.devices[deviceID]
	if !ok {
		return nil, errors.New("device not found")
	}

	deviceCopy := *device

	return &deviceCopy, nil
}

func (r *fakeOrgDevicesRepository) CreateDevice(
	ctx context.Context,
	device *domain.OrganizationDevice,
) error {
	_ = ctx
	deviceCopy := *device
	r.devices[device.ID] = &deviceCopy

	return nil
}

func (r *fakeOrgDevicesRepository) DeleteDevice(ctx context.Context, deviceID uuid.UUID) error {
	_ = ctx

	delete(r.devices, deviceID)

	return nil
}

func (r *fakeOrgDevicesRepository) SetName(
	ctx context.Context,
	deviceID uuid.UUID,
	name string,
) error {
	_ = ctx

	device, ok := r.devices[deviceID]
	if !ok {
		return errors.New("device not found")
	}

	device.Name = name

	return nil
}

func (r *fakeOrgDevicesRepository) DeviceExists(
	ctx context.Context,
	deviceID uuid.UUID,
) (bool, error) {
	_ = ctx
	_, ok := r.devices[deviceID]

	return ok, nil
}
