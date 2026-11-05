package messages

import "github.com/google/uuid"

type IdentificationHeader struct {
	DeviceID uuid.UUID `header:"device-id" validate:"required"` //nolint:tagliatelle
}
