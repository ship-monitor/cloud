//go:build integration

package device_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domain "github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/internal/v2/repositories/device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deviceExists reports whether a device row with the given id is still
// present. It queries the table directly so the assertion does not depend
// on another repository method under test.
func deviceExists(
	t *testing.T,
	ctx context.Context,
	db *sqlx.DB,
	deviceID uuid.UUID,
) bool {
	t.Helper()

	var count int

	err := db.GetContext(ctx, &count,
		`SELECT count(*) FROM devices WHERE id = $1`, deviceID)
	require.NoError(t, err)

	return count > 0
}

// deleteCase describes one Delete outcome.
type deleteCase struct {
	name          string
	foreignOwner  bool
	missingDevice bool
	wantErr       error
	wantDeleted   bool
}

// checkDeleteCase runs one Delete case against a freshly created device.
func checkDeleteCase(
	t *testing.T,
	ctx context.Context,
	repo *device.Repository,
	sqlDB *sqlx.DB,
	tt deleteCase,
) {
	t.Helper()

	stored := newDevice()
	require.NoError(t, repo.Create(ctx, stored))

	ownerID := stored.OwnerID
	if tt.foreignOwner {
		ownerID = uuid.New()
	}

	targetID := stored.ID
	if tt.missingDevice {
		targetID = uuid.New()
	}

	err := repo.Delete(ctx, ownerID, targetID)
	if tt.wantErr != nil {
		require.ErrorIs(t, err, tt.wantErr)
	} else {
		require.NoError(t, err)
	}

	if tt.wantDeleted {
		assert.False(t, deviceExists(t, ctx, sqlDB, stored.ID),
			"the device row must be removed")
	} else {
		assert.True(t, deviceExists(t, ctx, sqlDB, stored.ID),
			"the device row must remain")
	}
}

// TestRepository_Delete covers the delete outcomes. The security-relevant
// case is the foreign owner: deletion is scoped by owner_id, so a user
// must not be able to remove another user's device.
func TestRepository_Delete(t *testing.T) {
	t.Parallel()

	sqlDB := setupDB(t)
	repo := device.New(sqlDB)
	ctx := t.Context()

	cases := []deleteCase{
		{
			name:          "owned device",
			foreignOwner:  false,
			missingDevice: false,
			wantErr:       nil,
			wantDeleted:   true,
		},
		{
			name:          "device owned by another user",
			foreignOwner:  true,
			missingDevice: false,
			wantErr:       domain.ErrNotFound,
			wantDeleted:   false,
		},
		{
			name:          "missing device",
			foreignOwner:  false,
			missingDevice: true,
			wantErr:       domain.ErrNotFound,
			wantDeleted:   false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			checkDeleteCase(t, ctx, repo, sqlDB, tt)
		})
	}
}

// TestRepository_DeleteTwiceReportsNotFound verifies the error contract of
// a repeated delete: the first call removes the row, the second reports
// ErrNotFound instead of succeeding.
func TestRepository_DeleteTwiceReportsNotFound(t *testing.T) {
	t.Parallel()

	sqlDB := setupDB(t)
	repo := device.New(sqlDB)
	ctx := t.Context()

	stored := newDevice()
	require.NoError(t, repo.Create(ctx, stored))

	require.NoError(t, repo.Delete(ctx, stored.OwnerID, stored.ID))
	require.False(t, deviceExists(t, ctx, sqlDB, stored.ID))

	err := repo.Delete(ctx, stored.OwnerID, stored.ID)
	require.ErrorIs(t, err, domain.ErrNotFound,
		"deleting an already removed device must report not found")
}

// TestRepository_DeleteOnlyTargetsTheGivenDevice verifies that deleting
// one device leaves other devices untouched.
func TestRepository_DeleteOnlyTargetsTheGivenDevice(t *testing.T) {
	t.Parallel()

	sqlDB := setupDB(t)
	repo := device.New(sqlDB)
	ctx := t.Context()

	kept := newDevice()
	removed := newDevice()

	require.NoError(t, repo.Create(ctx, kept))
	require.NoError(t, repo.Create(ctx, removed))

	require.NoError(t, repo.Delete(ctx, removed.OwnerID, removed.ID))

	assert.False(t, deviceExists(t, ctx, sqlDB, removed.ID),
		"the targeted device must be removed")
	assert.True(t, deviceExists(t, ctx, sqlDB, kept.ID),
		"other devices must remain")
}
