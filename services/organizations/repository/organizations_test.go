package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
)

func createDB(t *testing.T) *sql.DB {
	// Setup in-memory database for testing
	db, err := sql.Open(sqliteshim.ShimName, ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	return db
}

func TestMigrate(t *testing.T) {
	db := createDB(t)
	repo := New(db)
	err := repo.Migrate(t.Context())
	if err != nil {
		t.Fatalf("Failed migrate with error: %s", err)
	}
}

func TestUserIsMember(t *testing.T) {
}

func TestGetUsersOrganizations(t *testing.T) {
	db := createDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create repository
	repo := New(db)
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("Failed migrate DB: %s", err)
	}

	// Initialize bun DB
	bunDB := bun.NewDB(db, sqlitedialect.New())

	// Create test data
	userID := uuid.New()
	orgID1 := uuid.New()
	orgID2 := uuid.New()

	// Insert test organizations
	now := time.Now()
	if _, err := bunDB.NewInsert().Model(&data.Organization{
		ID:        orgID1,
		Name:      "Test Org 1",
		CreatorID: userID,
		CreatedAt: now,
	}).Exec(ctx); err != nil {
		t.Fatalf("failed to insert organization 1: %v", err)
	}

	if _, err := bunDB.NewInsert().Model(&data.Organization{
		ID:        orgID2,
		Name:      "Test Org 2",
		CreatorID: userID,
		CreatedAt: now,
	}).Exec(ctx); err != nil {
		t.Fatalf("failed to insert organization 2: %v", err)
	}

	// Insert test members
	if _, err := bunDB.NewInsert().Model(&data.OrganizationMember{
		MemberID:       userID,
		OrganizationID: orgID1,
		JoinedAt:       now,
		Role:           data.RoleOwner,
	}).Exec(ctx); err != nil {
		t.Fatalf("failed to insert member 1: %v", err)
	}

	if _, err := bunDB.NewInsert().Model(&data.OrganizationMember{
		MemberID:       userID,
		OrganizationID: orgID2,
		JoinedAt:       now,
		Role:           data.RoleMember,
	}).Exec(ctx); err != nil {
		t.Fatalf("failed to insert member 2: %v", err)
	}

	// Test GetUsersOrganizations
	organizations, err := repo.GetUsersOrganizations(ctx, userID)
	if err != nil {
		t.Fatalf("GetUsersOrganizations failed: %v", err)
	}

	if len(organizations) != 2 {
		t.Errorf("expected 2 organizations, got %d", len(organizations))
	}

	// Verify the organizations
	orgIDs := make(map[uuid.UUID]bool)
	for _, org := range organizations {
		orgIDs[org.ID] = true
	}

	if !orgIDs[orgID1] {
		t.Error("expected organization 1 to be in results")
	}
	if !orgIDs[orgID2] {
		t.Error("expected organization 2 to be in results")
	}

	// User is in organization

	isMember, err := repo.UserIsMember(ctx, userID, orgID1)
	if err != nil {
		t.Fatalf("failed check user membership: %s", err)
	} else if !isMember {
		t.Fatal("user is not member, but should be")
	}
}
