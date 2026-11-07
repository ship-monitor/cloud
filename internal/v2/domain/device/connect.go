package device

import (
	"time"

	"github.com/google/uuid"
)

type ConnectRequest struct {
	ID        uuid.UUID
	DeviceID  uuid.UUID
	CreatedAt time.Time
	Approved  bool
	UserID    uuid.UUID
}
