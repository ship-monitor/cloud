package connection

import (
	"context"

	"github.com/google/uuid"
)

type HubID uuid.UUID
type DeviceID uuid.UUID

type PermissionService interface {
	CanSendData(ctx context.Context, hubID HubID, deviceID DeviceID) bool
}
