package device

import (
	"context"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/domain"
)

type StatesRepository interface {
	GetStates(
		ctx context.Context,
		deviceID, state string,
		historyLength int,
	) ([]domain.StateRecord, error)
}

type DeviceRepository interface {
	GetDevice(
		ctx context.Context,
		deviceID domain.DeviceID,
	) (*domain.Device, error)
	GetDevicesByIDs(
		ctx context.Context,
		deviceIDs []domain.DeviceID,
	) ([]domain.Device, error)
	ConnectDevice(
		ctx context.Context,
		deviceID domain.DeviceID,
		userID uuid.UUID,
		name string,
	) (*domain.Device, error)
	RenameDevice(
		ctx context.Context,
		deviceID domain.DeviceID,
		name string,
	) error
	DisconnectDevice(
		ctx context.Context,
		deviceID domain.DeviceID,
	) error
}
