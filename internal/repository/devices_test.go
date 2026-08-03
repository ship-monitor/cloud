package repository_test

import (
	"database/sql"
	"errors"
	"testing"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"github.com/uptrace/bun/driver/sqliteshim"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/repository"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/paging"
)

func createDB(t *testing.T) *sql.DB {
	t.Helper()
	// Setup in-memory database for testing
	db, err := sql.Open(sqliteshim.ShimName, ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	db.SetMaxOpenConns(1)

	return db
}

func closeDB(t *testing.T, db *sql.DB) {
	t.Helper()

	if err := db.Close(); err != nil {
		t.Errorf("failed to close database: %v", err)
	}
}

func TestDeviceRepoMigrate(t *testing.T) {
	t.Parallel()

	db := createDB(t)
	defer closeDB(t, db)

	repo := repository.NewDevices(db, log.Default())
	if err := repo.Migrate(t.Context()); err != nil {
		t.Fatalf("migrate devices: %v", err)
	}
}

func TestDeviceRepoGetDevicesByIDs(t *testing.T) {
	t.Parallel()

	db := createDB(t)
	defer closeDB(t, db)

	repo := repository.NewDevices(db, log.Default())
	if err := repo.Migrate(t.Context()); err != nil {
		t.Fatalf("migrate devices: %v", err)
	}

	deviceID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	secondDeviceID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	for _, id := range []domain.DeviceID{deviceID, secondDeviceID} {
		if err := repo.InsertDevice(t.Context(), &domain.Device{
			ID:           id,
			PasswordHash: []byte("password-hash"),
			Model:        testDeviceModel,
		}); err != nil {
			t.Fatalf("insert device %s: %v", id, err)
		}
	}

	available, err := repo.GetDevicesByIDs(
		t.Context(),
		[]domain.DeviceID{secondDeviceID},
	)
	if err != nil {
		t.Fatalf("get devices by IDs: %v", err)
	}

	if len(available) != 1 || available[0].ID != secondDeviceID {
		t.Fatalf("expected only device %s, got %+v", secondDeviceID, available)
	}

	empty, err := repo.GetDevicesByIDs(t.Context(), nil)
	if err != nil {
		t.Fatalf("get devices by empty ID list: %v", err)
	}

	if len(empty) != 0 {
		t.Fatalf("expected no devices, got %+v", empty)
	}
}

const (
	bridgeName      = "Bridge"
	testDeviceModel = "ship-monitor-v1"
)

func TestDeviceRepoMethods(t *testing.T) {
	t.Parallel()

	db := createDB(t)
	defer closeDB(t, db)

	repo := repository.NewDevices(db, log.Default())
	if err := repo.Migrate(t.Context()); err != nil {
		t.Fatalf("migrate devices: %v", err)
	}

	deviceID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	secondDeviceID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	if err := repo.InsertDevice(t.Context(), &domain.Device{
		ID:           deviceID,
		PasswordHash: []byte("password-hash"),
		Model:        testDeviceModel,
	}); err != nil {
		t.Fatalf("insert device: %v", err)
	}

	if err := repo.InsertDevice(t.Context(), &domain.Device{
		ID:           secondDeviceID,
		PasswordHash: []byte("second-password-hash"),
		Model:        "ship-monitor-v2",
	}); err != nil {
		t.Fatalf("insert second device: %v", err)
	}

	device, err := repo.GetDevice(t.Context(), deviceID)
	if err != nil {
		t.Fatalf("get device: %v", err)
	}

	if device.ID != deviceID {
		t.Fatalf("expected device id %s, got %s", deviceID, device.ID)
	}

	if device.Model != testDeviceModel {
		t.Fatalf("expected model %s, got %q", testDeviceModel, device.Model)
	}

	if device.OwnerID != nil {
		t.Fatalf("expected unowned device, got owner %s", *device.OwnerID)
	}

	if device.Name != nil {
		t.Fatalf("expected unnamed device, got %q", *device.Name)
	}

	devices, err := repo.GetDevices(
		t.Context(),
		paging.Paging{Page: 0, Size: 10},
	)
	if err != nil {
		t.Fatalf("get devices: %v", err)
	}

	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}

	page, err := repo.GetDevices(t.Context(), paging.Paging{Page: 0, Size: 1})
	if err != nil {
		t.Fatalf("get first page: %v", err)
	}

	if len(page) != 1 || page[0].ID != deviceID {
		t.Fatalf("expected first page to contain %s, got %+v", deviceID, page)
	}

	ownerID := uuid.New()

	connected, err := repo.ConnectDevice(
		t.Context(),
		deviceID,
		ownerID,
		bridgeName,
	)
	if err != nil {
		t.Fatalf("connect device: %v", err)
	}

	if connected.OwnerID == nil || *connected.OwnerID != ownerID {
		t.Fatalf(
			"expected connected owner %s, got %+v",
			ownerID,
			connected.OwnerID,
		)
	}

	if connected.Name == nil || *connected.Name != bridgeName {
		t.Fatalf(
			"expected connected device name %s, got %+v",
			bridgeName,
			connected.Name,
		)
	}

	_, err = repo.ConnectDevice(
		t.Context(),
		deviceID,
		uuid.New(),
		"Replacement",
	)
	if !errors.Is(err, domain.ErrDeviceAlreadyConnected) {
		t.Fatalf("reconnect device: expected already connected, got %v", err)
	}

	if err := repo.RenameDevice(
		t.Context(),
		deviceID,
		"Engine Room",
	); err != nil {
		t.Fatalf("rename device: %v", err)
	}

	device, err = repo.GetDevice(t.Context(), deviceID)
	if err != nil {
		t.Fatalf("get renamed device: %v", err)
	}

	if device.Name == nil || *device.Name != "Engine Room" {
		t.Fatalf(
			"expected renamed device name Engine Room, got %+v",
			device.Name,
		)
	}

	if err := repo.DisconnectDevice(t.Context(), deviceID); err != nil {
		t.Fatalf("disconnect device: %v", err)
	}

	device, err = repo.GetDevice(t.Context(), deviceID)
	if err != nil {
		t.Fatalf("get disconnected device: %v", err)
	}

	if device.OwnerID != nil {
		t.Fatalf(
			"expected disconnected device owner to be nil, got %s",
			*device.OwnerID,
		)
	}

	if device.Name != nil {
		t.Fatalf(
			"expected disconnected device name to be nil, got %q",
			*device.Name,
		)
	}

	_, err = repo.GetDevice(t.Context(), uuid.New())
	if err == nil {
		t.Fatal("expected missing device lookup to fail")
	}
}
