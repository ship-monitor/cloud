package notification

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound is returned when no notification matches the given constraints.
var ErrNotFound = errors.New("notification not found")

// Notification represents a device notification.
type Notification struct {
	ID         uuid.UUID
	DeviceID   uuid.UUID
	SensorID   *uuid.UUID
	Type       string
	Severity   string
	OccurredAt time.Time
	ReportedAt time.Time
	ReceivedAt time.Time
	Data       map[string]any
}
