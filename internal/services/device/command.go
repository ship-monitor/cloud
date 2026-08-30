package device

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Command struct {
	Command string `json:"command"`
	Args    any    `json:"args"`
}

func getDeviceCommandTopic(deviceID uuid.UUID) string {
	return fmt.Sprintf("devices.%s.commands", deviceID.String())
}

func (s *Service) SendCommand(
	ctx context.Context,
	deviceID, userID uuid.UUID,
	command string,
	args any,
) error {
	if err := s.checkPermissions(
		ctx,
		deviceID,
		userID,
		DevicePermissionSendCommand,
	); err != nil {
		return fmt.Errorf("check permissions: %w", err)
	}

	cmd := Command{
		Command: command,
		Args:    args,
	}

	s.logger.Info(
		"Sending command",
		"deviceID", deviceID,
		"command", command,
		"topic", getDeviceCommandTopic(deviceID),
	)

	err := s.topicPublisher.PublishJSON(
		ctx,
		getDeviceCommandTopic(deviceID),
		cmd,
	)
	if err != nil {
		return fmt.Errorf("publish command: %w", err)
	}

	return nil
}
