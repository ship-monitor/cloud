package data

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/connector/connections"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/dto"
)

const DefaultDeviceName = "Unknown Device"

type OrganizationDevice struct {
	*bun.BaseModel `bun:"table:organization_devices"`

	ID             uuid.UUID `bun:",pk,type:varchar" json:"id"`
	OrganizationID uuid.UUID `bun:",notnull,type:varchar" json:"organizationId"`
	CreatedAt      time.Time `bun:",nullzero,notnull" json:"createdAt"`
	Name           string    `bun:",notnull,default" json:"name"`
}

func ConnectDevice(id, organizationID uuid.UUID, name string) error {
	if device, _ := GetDevice(id); device != nil {
		if device.OrganizationID != organizationID {
			return fmt.Errorf("device %q is already connected to another organization", id)
		}
		return nil
	}

	if name == "" {
		name = GenNodeName(id)
	}
	if connections.IsConnected(id) {
		_, err := createDevice(id, organizationID, name)
		return err
	}
	return fmt.Errorf("device %q is not connected connected to server", id)
}

func createDevice(id, orgID uuid.UUID, name string) (*OrganizationDevice, error) {
	device := OrganizationDevice{
		ID:             id,
		OrganizationID: orgID,
		CreatedAt:      time.Now(),
		Name:           name,
	}
	_, err := db.DB.NewInsert().Model(&device).Exec(context.Background())
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (device *OrganizationDevice) toDTO() *dto.DeviceResponse {
	return &dto.DeviceResponse{
		ID:             device.ID,
		OrganizationID: device.OrganizationID,
		CreatedAt:      device.CreatedAt,
		IsConnected:    connections.IsConnected(device.ID),
		Name:           device.Name,
	}
}

func GetDevice(id uuid.UUID) (*dto.DeviceResponse, error) {
	var device OrganizationDevice
	err := db.DB.NewSelect().
		Model(&device).
		Where("id = ?", id).
		Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return device.toDTO(), nil
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
