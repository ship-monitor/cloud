package services

import (
	"context"
	"errors"
	"fmt"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
)

type DevicesService struct {
	states     *DeviceStatesService
	orgDevices *OrgDevicesService
	logger     *log.Logger
	orgs       *OrganizationsService
}

func NewDevices(
	states *DeviceStatesService,
	orgDevices *OrgDevicesService,
	logger *log.Logger,
	orgs *OrganizationsService,
) *DevicesService {
	return &DevicesService{
		states:     states,
		orgDevices: orgDevices,
		logger:     logger,
		orgs:       orgs,
	}
}

var (
	ErrUserCantGetState     = errors.New("user can't get device state")
	ErrInvalidHistoryLength = errors.New("invalid history length specified")
)

func (d *DevicesService) GetStates(
	ctx context.Context,
	deviceID, userID uuid.UUID,
	state string,
	historyLength int,
) ([]domain.StateRecord, error) {
	if historyLength < 0 {
		return nil, ErrInvalidHistoryLength
	}

	if can, err := d.orgDevices.UserCanGetState(ctx, userID, deviceID); err != nil {
		return nil, fmt.Errorf("check user can get state: %w", err)
	} else if !can {
		return nil, ErrUserCantGetState
	}

	states, err := d.states.GetStates(ctx, deviceID.String(), state, historyLength)
	if err != nil {
		return nil, fmt.Errorf("get states: %w", err)
	}

	return states, err
}
