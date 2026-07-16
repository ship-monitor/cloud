package repository_test

import (
	"testing"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/repository"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/paging"
)

func TestDeviceRepoMigrate(t *testing.T) {
	t.Parallel()

	db := createDB(t)

	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	}()

	repo := repository.NewDevices(db, log.Default())
	if err := repo.Migrate(t.Context()); err != nil {
		t.Fatalf("migrate devices: %v", err)
	}
}

func TestDeviceRepoMethods(t *testing.T) {
	t.Parallel()

	const bridgeName = "Bridge"

	db := createDB(t)

	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	}()

	ctx := t.Context()

	repo := repository.NewDevices(db, log.Default())
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("migrate devices: %v", err)
	}

	deviceID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	secondDeviceID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	if err := repo.InsertDevice(ctx, &domain.Device{
		ID:           deviceID,
		PasswordHash: []byte("password-hash"),
		Model:        "ship-monitor-v1",
	}); err != nil {
		t.Fatalf("insert device: %v", err)
	}

	if err := repo.InsertDevice(ctx, &domain.Device{
		ID:           secondDeviceID,
		PasswordHash: []byte("second-password-hash"),
		Model:        "ship-monitor-v2",
	}); err != nil {
		t.Fatalf("insert second device: %v", err)
	}

	device, err := repo.GetDevice(ctx, deviceID)
	if err != nil {
		t.Fatalf("get device: %v", err)
	}

	if device.ID != deviceID {
		t.Fatalf("expected device id %s, got %s", deviceID, device.ID)
	}

	if device.Model != "ship-monitor-v1" {
		t.Fatalf("expected model ship-monitor-v1, got %q", device.Model)
	}

	if device.OwnerID != nil {
		t.Fatalf("expected unowned device, got owner %s", *device.OwnerID)
	}

	if device.Name != nil {
		t.Fatalf("expected unnamed device, got %q", *device.Name)
	}

	devices, err := repo.GetDevices(ctx, paging.Paging{Page: 0, Size: 10})
	if err != nil {
		t.Fatalf("get devices: %v", err)
	}

	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}

	page, err := repo.GetDevices(ctx, paging.Paging{Page: 0, Size: 1})
	if err != nil {
		t.Fatalf("get first page: %v", err)
	}

	if len(page) != 1 || page[0].ID != deviceID {
		t.Fatalf("expected first page to contain %s, got %+v", deviceID, page)
	}

	ownerID := uuid.New()

	connected, err := repo.ConnectDevice(ctx, deviceID, ownerID, bridgeName)
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

	if err := repo.RenameDevice(ctx, deviceID, "Engine Room"); err != nil {
		t.Fatalf("rename device: %v", err)
	}

	device, err = repo.GetDevice(ctx, deviceID)
	if err != nil {
		t.Fatalf("get renamed device: %v", err)
	}

	if device.Name == nil || *device.Name != "Engine Room" {
		t.Fatalf(
			"expected renamed device name Engine Room, got %+v",
			device.Name,
		)
	}

	if err := repo.DisconnectDevice(ctx, deviceID); err != nil {
		t.Fatalf("disconnect device: %v", err)
	}

	device, err = repo.GetDevice(ctx, deviceID)
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

	_, err = repo.GetDevice(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected missing device lookup to fail")
	}
}
