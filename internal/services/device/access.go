package device

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/domain"
)

func (s *Service) SetDeviceAccess(
	ctx context.Context,
	applicant *domain.Principal,
	deviceID uuid.UUID,
	access domain.DeviceAccess,
) error {
	relation, err := deviceRelationForAccessRole(access.Role)
	if err != nil {
		return fmt.Errorf("resolve device access role: %w", err)
	}

	if err := s.checkPermissions(
		ctx,
		deviceID,
		applicant.UserID,
		DevicePermissionShare,
	); err != nil {
		return fmt.Errorf("check permission: %w", err)
	}

	if err := s.setAccessRelation(
		ctx,
		deviceID,
		access.UserID,
		relation,
	); err != nil {
		return fmt.Errorf("set access relation: %w", err)
	}

	return nil
}

func (s *Service) DeleteDeviceAccess(
	ctx context.Context,
	applicant *domain.Principal,
	deviceID, userID uuid.UUID,
) error {
	if err := s.checkPermissions(
		ctx,
		deviceID,
		applicant.UserID,
		DevicePermissionShare,
	); err != nil {
		return fmt.Errorf("check permission: %w", err)
	}

	if err := s.deleteAccessRelations(ctx, deviceID, userID); err != nil {
		return fmt.Errorf("delete access relations: %w", err)
	}

	return nil
}

func (s *Service) GetDeviceAccess(
	ctx context.Context,
	applicant *domain.Principal,
	deviceID uuid.UUID,
) ([]domain.DeviceAccess, error) {
	if err := s.checkPermissions(
		ctx,
		deviceID,
		applicant.UserID,
		DevicePermissionShare,
	); err != nil {
		return nil, fmt.Errorf("check permission: %w", err)
	}

	access, err := s.readDeviceAccess(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("read device access: %w", err)
	}

	return access, nil
}
