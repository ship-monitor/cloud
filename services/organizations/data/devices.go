package data

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/connections"
)

type OrganizationDevice struct {
	*bun.BaseModel `bun:"table:organization_devices"`

	ID             uuid.UUID `bun:",pk,type:varchar" json:"id"`
	OrganizationID uuid.UUID `bun:",notnull,type:varchar" json:"organizationId"`
	CreatedAt      time.Time `bun:",nullzero,notnull" json:"createdAt"`
}

func ConnectDevice(id, organizationID uuid.UUID) error {
	if device, _ := GetDevice(id); device != nil {
		if device.OrganizationID != organizationID {
			return fmt.Errorf("device %q is already connected to another organization", id)
		}
		return nil
	}
	if connections.IsConnected(id) {
		_, err := createDevice(id, organizationID)
		return err
	}
	return fmt.Errorf("device %q is not connected connected to server", id)
}

func createDevice(id, orgID uuid.UUID) (*OrganizationDevice, error) {
	device := OrganizationDevice{
		ID:             id,
		OrganizationID: orgID,
		CreatedAt:      time.Now(),
	}
	_, err := db.DB.NewInsert().Model(&device).Exec(context.Background())
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func GetDevice(id uuid.UUID) (*OrganizationDevice, error) {
	var device OrganizationDevice
	err := db.DB.NewSelect().
		Model(&device).
		Where("id = ?", id).
		Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func ListDevices(orgID uuid.UUID) ([]OrganizationDevice, error) {
	var devices []OrganizationDevice
	err := db.DB.NewSelect().
		Model(&devices).
		Where("organization_id = ?", orgID).
		Order("created_at ASC").
		Scan(context.Background())
	return devices, err
}

func DisconnectDevice(id uuid.UUID) error {
	if device, _ := GetDevice(id); device != nil {
		return deleteDevice(id)
	}
	return nil
}

func deleteDevice(id uuid.UUID) error {
	_, err := db.DB.NewDelete().
		Model((*OrganizationDevice)(nil)).
		Where("id = ?", id).
		Exec(context.Background())
	return err
}
