package sensor

import (
	"github.com/google/uuid"
)

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

// SensorDecl issensor declaration.
type SensorDecl struct {
	ID           ID
	Name         string
	Manufacturer string
	Model        string
	Revision     string
	Software     string
}

type ID = uuid.UUID
