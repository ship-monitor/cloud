package domain

import "github.com/google/uuid"

type ConnectDeviceRequest struct {
	DeviceID  uuid.UUID
	DeviceKey string
	HubID     uuid.UUID
}
