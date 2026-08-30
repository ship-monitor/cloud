package device

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/domain"
)

func (s *Service) GetStates(
	ctx context.Context,
	deviceID, userID uuid.UUID,
	state string,
	historyLength int,
) ([]domain.StateRecord, error) {
	if err := s.checkPermissions(
		ctx,
		deviceID,
		userID,
		DevicePermissionViewState,
	); err != nil {
		return nil, fmt.Errorf("check permissions: %w", err)
	}

	if historyLength < 0 {
		return nil, ErrInvalidHistoryLength
	}

	states, err := s.states.GetStates(
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
