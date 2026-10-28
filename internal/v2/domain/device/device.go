// Package device contains the domain model for devices.
package device

import "github.com/google/uuid"

// Device is a ship device owned by a user.
type Device struct {
	ID          uuid.UUID
	OwnerID     uuid.UUID
	Name        string
	SecurityKey string
}
