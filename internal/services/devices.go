package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/names"
)

type StatesRepository interface {
	GetStates(
		ctx context.Context,
		deviceID, state string,
		historyLength int,
	) ([]domain.StateRecord, error)
}

type DeviceRepository interface {
	GetDevice(
		ctx context.Context,
		deviceID domain.DeviceID,
	) (*domain.Device, error)
	ConnectDevice(
		ctx context.Context,
		deviceID domain.DeviceID,
		userID uuid.UUID,
		name string,
	) (*domain.Device, error)
}

type DevicesService struct {
	devices        DeviceRepository
	states         StatesRepository
	logger         *log.Logger
	topicPublisher *TopicPublisher
}

func NewDevices(
	states StatesRepository,
	devices DeviceRepository,
	logger *log.Logger,
	topicPublisher *TopicPublisher,
) *DevicesService {
	return &DevicesService{
		devices:        devices,
		states:         states,
		logger:         logger,
		topicPublisher: topicPublisher,
	}
}

var (
	ErrForbidden             = errors.New("this action forbidden")
	ErrInvalidHistoryLength  = errors.New("invalid history length specified")
	ErrDeviceNotFound        = errors.New("device not found")
	ErrInvalidDevicePassword = errors.New("invalid device password")
	ErrAlreadyConnected      = domain.ErrDeviceAlreadyConnected
)

func (d *DevicesService) ConnectDevice(
	ctx context.Context,
	deviceID, userID uuid.UUID,
	password, name string,
) error {
	device, err := d.devices.GetDevice(ctx, deviceID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrDeviceNotFound
	case err != nil:
		return fmt.Errorf("get device: %w", err)
	case device.OwnerID != nil:
		return ErrAlreadyConnected
	case !device.CheckPassword(password):
		return ErrInvalidDevicePassword
	}

	if name == "" {
		name = names.Gen()
	}

	_, err = d.devices.ConnectDevice(ctx, deviceID, userID, name)
	switch {
	case errors.Is(err, domain.ErrDeviceAlreadyConnected):
		return ErrAlreadyConnected
	case err != nil:
		return fmt.Errorf("connect device: %w", err)
	default:
		return nil
	}
}

func (d *DevicesService) GetStates(
	ctx context.Context,
	deviceID, userID uuid.UUID,
	state string,
	historyLength int,
) ([]domain.StateRecord, error) {
	if historyLength < 0 {
		return nil, ErrInvalidHistoryLength
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
	return fmt.Sprintf("devices.%s.commands", deviceID.String())
}
