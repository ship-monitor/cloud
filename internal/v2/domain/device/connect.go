package device

import (
	"errors"
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

// ErrNotFound is returned when the device does not exist,
// or exists but is owned by another user.
var ErrNotFound = errors.New("device not found")

// ErrAlreadyExists is returned when a device with the given id
// was already added.
var ErrAlreadyExists = errors.New("device already exists")
