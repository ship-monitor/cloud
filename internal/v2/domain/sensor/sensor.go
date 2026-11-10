package sensor

import (
	"time"

	"github.com/google/uuid"
)

type Sensor struct {
	ID       string    `json:"id" validate:"required"`
	Name     string    `json:"name" validate:"required"`
	DeviceID uuid.UUID `json:"deviceId" validate:"required"`
}

type Container struct {
	Sensors      []Sensor  `json:"sensors"`
	CreationDate time.Time `json:"creation"`
}

func NewContainer(sensors []Sensor) *Container {
	return &Container{
		Sensors:      sensors,
		CreationDate: time.Now().UTC(),
	}
}
