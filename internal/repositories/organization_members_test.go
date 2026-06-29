package repository_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	repository "sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/repositories"
)

func TestOrganizationsRepoMemberMethods(t *testing.T) {
	t.Parallel()

	db := createDB(t)

	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	}()

	ctx := t.Context()
	orgs := repository.NewOrgs(db)
	bunDB := bun.NewDB(db, sqlitedialect.New())

	if _, err := bunDB.NewCreateTable().
		Model((*domain.User)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		t.Fatalf("migrate users: %v", err)
	}

	if err := orgs.Migrate(ctx); err != nil {
		t.Fatalf("migrate organizations: %v", err)
	}

	ownerID := uuid.New()
	memberID := uuid.New()

	if _, err := bunDB.NewInsert().
		Model(testUser(ownerID, "Owner", "owner@example.com")).
		Exec(ctx); err != nil {
		t.Fatalf("create owner user: %v", err)
	}

	if _, err := bunDB.NewInsert().
		Model(testUser(memberID, "Member", "member@example.com")).
		Exec(ctx); err != nil {
		t.Fatalf("create member user: %v", err)
	}

	orgID, err := orgs.CreateOrganization(ctx, "Acme", ownerID)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}

	joinedAt := time.Now().Add(time.Minute)
	if err := orgs.AddMember(ctx, &domain.OrganizationMember{
		MemberID:       memberID,
		OrganizationID: orgID,
		Role:           domain.RoleAdministrator,
		JoinedAt:       joinedAt,
	}); err != nil {
		t.Fatalf("add member: %v", err)
	}

	member, err := orgs.GetMember(ctx, orgID, memberID)
	if err != nil {
		t.Fatalf("get member: %v", err)
	}

	if member.Role != domain.RoleAdministrator {
		t.Fatalf("expected administrator role, got %q", member.Role)
	}

	isMember, err := orgs.UserIsMember(ctx, memberID, orgID)
	if err != nil {
		t.Fatalf("check member: %v", err)
	}

	if !isMember {
		t.Fatal("expected user to be organization member")
	}

	if err := orgs.UpdateMemberRole(
		ctx,
		orgID,
		memberID,
		domain.RoleMember,
	); err != nil {
		t.Fatalf("update member role: %v", err)
	}

	member, err = orgs.GetMember(ctx, orgID, memberID)
	if err != nil {
		t.Fatalf("get updated member: %v", err)
	}

	if member.Role != domain.RoleMember {
		t.Fatalf("expected member role, got %q", member.Role)
	}

	members, err := orgs.GetMembersWithUserInfo(ctx, orgID)
	if err != nil {
		t.Fatalf("get members with user info: %v", err)
	}

	if len(members) != 2 {
		t.Fatalf("expected owner and member, got %d members", len(members))
	}

	membersByID := make(
		map[uuid.UUID]domain.OrganizationMemberWithUser,
		len(members),
	)
	for _, member := range members {
		membersByID[member.MemberID] = member
	}

	if membersByID[memberID].Email != "member@example.com" {
		t.Fatalf(
			"expected joined member email, got %q",
			membersByID[memberID].Email,
		)
	}

	if err := orgs.RemoveMember(ctx, orgID, memberID); err != nil {
		t.Fatalf("remove member: %v", err)
	}

	isMember, err = orgs.UserIsMember(ctx, memberID, orgID)
	if err != nil {
		t.Fatalf("check removed member: %v", err)
	}

	if isMember {
		t.Fatal("expected user to be removed from organization")
	}

	if err := orgs.DeleteOrganization(ctx, orgID); err != nil {
		t.Fatalf("delete organization: %v", err)
	}

	_, err = orgs.GetOrganizationByID(ctx, orgID)
	if err == nil {
		t.Fatal("expected deleted organization lookup to fail")
	}

	ownerIsMember, err := orgs.UserIsMember(ctx, ownerID, orgID)
	if err != nil {
		t.Fatalf("check owner membership after delete: %v", err)
	}

	if ownerIsMember {
		t.Fatal("expected organization members to be deleted with organization")
	}
}

func TestOrganizationsRepoInvitationMethods(t *testing.T) {
	t.Parallel()

	db := createDB(t)

	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	}()

	ctx := t.Context()

	orgs := repository.NewOrgs(db)
	if err := orgs.Migrate(ctx); err != nil {
		t.Fatalf("migrate organizations: %v", err)
	}

	ownerID := uuid.New()

	orgID, err := orgs.CreateOrganization(ctx, "Acme", ownerID)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour)

	inv, err := orgs.CreateInvitation(
		ctx,
		orgID,
		"invitee@example.com",
		expiresAt,
	)
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}

	if inv.ID == uuid.Nil {
		t.Fatal("expected generated invitation id")
	}

	if inv.Status != domain.InvStatusPending {
		t.Fatalf("expected pending status, got %q", inv.Status)
	}

	exists, err := orgs.HasPendingInvitation(ctx, orgID, "invitee@example.com")
	if err != nil {
		t.Fatalf("check pending invitation: %v", err)
	}

	if !exists {
		t.Fatal("expected pending invitation to exist")
	}

	byID, err := orgs.GetInvitationByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("get invitation by id: %v", err)
	}

	if byID.InviteeEmail != "invitee@example.com" {
		t.Fatalf("expected invitee email, got %q", byID.InviteeEmail)
	}

	userInvitations, err := orgs.ListInvitationsForUser(
		ctx,
		"invitee@example.com",
	)
	if err != nil {
		t.Fatalf("list user invitations: %v", err)
	}

	if len(userInvitations) != 1 {
		t.Fatalf("expected 1 user invitation, got %d", len(userInvitations))
	}

	orgInvitations, err := orgs.ListInvitationsForOrg(ctx, orgID)
	if err != nil {
		t.Fatalf("list organization invitations: %v", err)
	}

	if len(orgInvitations) != 1 {
		t.Fatalf(
			"expected 1 organization invitation, got %d",
			len(orgInvitations),
		)
	}

	if err := orgs.UpdateInvitationStatus(
		ctx,
		inv.ID,
		domain.InvStatusAccepted,
	); err != nil {
		t.Fatalf("update invitation status: %v", err)
	}

	byID, err = orgs.GetInvitationByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("get updated invitation: %v", err)
	}

	if byID.Status != domain.InvStatusAccepted {
		t.Fatalf("expected accepted status, got %q", byID.Status)
	}

	exists, err = orgs.HasPendingInvitation(ctx, orgID, "invitee@example.com")
	if err != nil {
		t.Fatalf("check accepted invitation pending state: %v", err)
	}

	if exists {
		t.Fatal("expected accepted invitation to no longer be pending")
	}

	userInvitations, err = orgs.ListInvitationsForUser(
		ctx,
		"invitee@example.com",
	)
	if err != nil {
		t.Fatalf("list user invitations after accept: %v", err)
	}

	if len(userInvitations) != 0 {
		t.Fatalf(
			"expected accepted invitation to be hidden from user pending list, got %d",
			len(userInvitations),
		)
	}
}

func TestOrgDevicesRepo(t *testing.T) {
	t.Parallel()

	db := createDB(t)

	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	}()

	ctx := t.Context()
	orgs := repository.NewOrgs(db)
	devices := repository.NewOrgDevices(db)

	if err := orgs.Migrate(ctx); err != nil {
		t.Fatalf("migrate organizations: %v", err)
	}

	orgID, err := orgs.CreateOrganization(ctx, "Acme", uuid.New())
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}

	deviceID := uuid.New()
	createdAt := time.Now().UTC().Truncate(time.Second)

	if err := devices.CreateDevice(ctx, &domain.OrganizationDevice{
		ID:             deviceID,
		OrganizationID: orgID,
		Name:           "Bridge",
		CreatedAt:      createdAt,
	}); err != nil {
		t.Fatalf("create device: %v", err)
	}

	exists, err := devices.DeviceExists(ctx, deviceID)
	if err != nil {
		t.Fatalf("check device exists: %v", err)
	}

	if !exists {
		t.Fatal("expected device to exist")
	}

	device, err := devices.GetDevice(ctx, deviceID)
	if err != nil {
		t.Fatalf("get device: %v", err)
	}

	if device.Name != "Bridge" || device.OrganizationID != orgID {
		t.Fatalf("unexpected device data: %+v", device)
	}

	listed, err := devices.ListDevices(ctx, orgID)
	if err != nil {
		t.Fatalf("list devices: %v", err)
	}

	if len(listed) != 1 || listed[0].ID != deviceID {
		t.Fatalf("expected listed device %s, got %+v", deviceID, listed)
	}

	if err := devices.SetName(ctx, deviceID, "Engine Room"); err != nil {
		t.Fatalf("set device name: %v", err)
	}

	device, err = devices.GetDevice(ctx, deviceID)
	if err != nil {
		t.Fatalf("get renamed device: %v", err)
	}

	if device.Name != "Engine Room" {
		t.Fatalf("expected renamed device, got %q", device.Name)
	}

	if err := devices.DeleteDevice(ctx, deviceID); err != nil {
		t.Fatalf("delete device: %v", err)
	}

	exists, err = devices.DeviceExists(ctx, deviceID)
	if err != nil {
		t.Fatalf("check deleted device exists: %v", err)
	}

	if exists {
		t.Fatal("expected device to be deleted")
	}

	_, err = devices.GetDevice(ctx, deviceID)
	if err == nil {
		t.Fatal("expected deleted device lookup to fail")
	}
}

func testUser(id uuid.UUID, name, email string) *domain.User {
	now := time.Now()

	return &domain.User{
		ID:            id,
		Name:          name,
		Email:         email,
		PasswordHash:  []byte("password-hash"),
		CreatedAt:     now,
		UpdatedAt:     now,
		Blocked:       false,
		EmailVerified: false,
	}
}
