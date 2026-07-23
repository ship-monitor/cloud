package domain

import (
	"errors"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

var ErrDeviceAlreadyConnected = errors.New("device already connected")

// DeviceID is the unique identifier for a device. Alias for [uuid.UUID].
type (
	DeviceID = uuid.UUID
	Device   struct {
		*bun.BaseModel `bun:"table:devices"`

		ID DeviceID `bun:",pk,type:varchar" json:"id"`
		// Hash of password for connecting device to cloud or connecting device
		// to
		// user. Compare passwords with [Device.CheckPassword]
		PasswordHash []byte `bun:",notnull" json:"-"`

		Model   string     `bun:",notnull" json:"model"`
		OwnerID *uuid.UUID `bun:",nullzero,type:varchar" json:"ownerId"`
		Name    *string    `bun:",nullzero" json:"name"`
	}
)

func (d *Device) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword(
		d.PasswordHash,
		[]byte(password),
	) == nil
}

func HashPassword(password string) []byte {
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil
	}

	return hash
}
