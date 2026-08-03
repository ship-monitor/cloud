package domain

import (
	"errors"

	"github.com/google/uuid"
)

type DeviceAccessRole string

var ErrInvalidDeviceAccessRole = errors.New("invalid device access role")

const (
	DeviceAccessRoleAdmin  DeviceAccessRole = "admin"
	DeviceAccessRoleViewer DeviceAccessRole = "viewer"
)

func (r DeviceAccessRole) Valid() bool {
	return r == DeviceAccessRoleAdmin || r == DeviceAccessRoleViewer
}

type DeviceAccess struct {
	UserID uuid.UUID        `json:"userId"`
	Role   DeviceAccessRole `json:"role"`
}
