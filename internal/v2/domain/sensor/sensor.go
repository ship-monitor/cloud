package sensor

import (
	"errors"

	"github.com/google/uuid"
)

// ErrNotFound is returned when no sensor matches the given ID and device.
var ErrNotFound = errors.New("sensor not found")

type Sensor struct {
	ID           ID        `json:"id" validate:"required"`
	Name         string    `json:"name" validate:"required"`
	DeviceID     uuid.UUID `json:"deviceId" validate:"required"`
	DisplayName  string    `json:"displayName" validate:"required"`
	Manufacturer string
	Model        string
	Revision     string
	Software     string
}

// SensorDecl is sensor declaration.
type SensorDecl struct {
	ID           ID
	Name         string
	Manufacturer string
	Model        string
	Revision     string
	Software     string
}

type ID = uuid.UUID

type SensorIdentity struct {
	SensorID ID        `validate:"required"`
	DeviceID uuid.UUID `validate:"required"`
}
