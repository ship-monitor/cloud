package device

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/ship-monitor/cloud/internal/domain"
	"github.com/ship-monitor/cloud/internal/handlers"
)

func (s *Service) ConnectDevice(
	ctx context.Context,
	applicant *domain.Principal,
	in handlers.ConnectDeviceIn,
) error {
	if err := validator.New().Struct(&in); err != nil {
		return fmt.Errorf("invalid input data: %w", err)
	}

	device, err := s.devices.GetDevice(ctx, in.DeviceID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return handlers.ErrDeviceNotFound
	case err != nil:
		return fmt.Errorf("get device: %w", err)
	case device.OwnerID != nil:
		return fmt.Errorf(
			"connect device: %w",
			domain.ErrDeviceAlreadyConnected,
		)
	case !device.CheckPassword(in.Password):
		return handlers.ErrInvalidDevicePassword
	}

	if err := s.addRelation(
		ctx,
		in.DeviceID,
		applicant.UserID,
		DeviceRelationOwner,
	); err != nil {
		return fmt.Errorf("add relation: %w", err)
	}

	_, err = s.devices.ConnectDevice(
		ctx,
		in.DeviceID,
		applicant.UserID,
		in.Name,
	)
	switch {
	case errors.Is(err, domain.ErrDeviceAlreadyConnected):
		return fmt.Errorf(
			"connect device: %w",
			domain.ErrDeviceAlreadyConnected,
		)
	case err != nil:
		return fmt.Errorf("connect device: %w", err)
	default:
		return nil
	}
}

func (s *Service) DisconnectDevice(
	ctx context.Context,
	deviceID domain.DeviceID,
	applicant *domain.Principal,
) error {
	if err := s.checkPermissions(
		ctx,
		deviceID,
		applicant.UserID,
		DevicePermissionDisconnect,
	); err != nil {
		return fmt.Errorf("check permission: %w", err)
	}

	if err := s.devices.DisconnectDevice(ctx, deviceID); err != nil {
		return fmt.Errorf("repo disconnect device: %w", err)
	}

	if err := s.clearRelations(ctx, deviceID); err != nil {
		return fmt.Errorf("clear permissions: %w", err)
	}

	return nil
}
