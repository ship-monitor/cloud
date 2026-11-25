package notification

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound is returned when no notification matches the given constraints.
var ErrNotFound = errors.New("notification not found")

// Cursor identifies a position in descending occurrence time and ID order.
// Callers should pass it back unchanged to continue a listing.
type Cursor struct {
	OccurredAt time.Time
	ID         uuid.UUID
}

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
	// IsRead is populated by user-scoped notification queries.
	IsRead bool
	Data   map[string]any
}
