package services

import (
	"context"
	"errors"
	"fmt"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
)

type StatesRepository interface {
	GetStates(
		ctx context.Context,
		deviceID, state string,
		historyLength int,
	) ([]domain.StateRecord, error)
}

type DevicesService struct {
	states         StatesRepository
	orgDevices     *OrgDevicesService
	logger         *log.Logger
	orgs           *OrganizationsService
	topicPublisher *TopicPublisher
}

func NewDevices(
	states StatesRepository,
	orgDevices *OrgDevicesService,
	logger *log.Logger,
	orgs *OrganizationsService,
	topicPublisher *TopicPublisher,
) *DevicesService {
	return &DevicesService{
		states:         states,
		orgDevices:     orgDevices,
		logger:         logger,
		orgs:           orgs,
		topicPublisher: topicPublisher,
	}
}

var (
	ErrForbidden            = errors.New("this action forbidden")
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

	if can, err := d.orgDevices.UserCanGetState(
		ctx,
		userID,
		deviceID,
	); err != nil {
		return nil, fmt.Errorf("check user can get state: %w", err)
	} else if !can {
		return nil, fmt.Errorf("user can't get state: %w", ErrForbidden)
	}

	states, err := d.states.GetStates(
		ctx,
		deviceID.String(),
		state,
		historyLength,
	)
	if err != nil {
		return nil, fmt.Errorf("get states: %w", err)
	}

	return states, nil
}

func (d *DevicesService) SendCommand(
	ctx context.Context,
	deviceID, userID uuid.UUID,
	command string,
	args any,
) error {
	if can, err := d.orgDevices.UserCanSendCommand(
		ctx,
		userID,
		deviceID,
	); err != nil {
		return fmt.Errorf("check user can send command: %w", err)
	} else if !can {
		return fmt.Errorf("user can't get state: %w", ErrForbidden)
	}

	cmd := Command{
		Command: command,
		Args:    args,
	}

	d.logger.Info(
		"Sending command",
		"deviceID", deviceID,
		"command", command,
		"topic", getDeeviceCommandTopic(deviceID),
	)

	err := d.topicPublisher.PublishJSON(
		ctx,
		getDeeviceCommandTopic(deviceID),
		cmd,
	)
	if err != nil {
		return fmt.Errorf("publish command: %w", err)
	}

	return nil
}

type Command struct {
	Command string `json:"command"`
	Args    any    `json:"args"`
}

func getDeeviceCommandTopic(deviceID uuid.UUID) string {
	return fmt.Sprintf("devices/%s/commands", deviceID.String())
}
