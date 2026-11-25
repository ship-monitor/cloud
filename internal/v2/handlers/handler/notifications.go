package handler

import (
	"context"

	"github.com/ship-monitor/cloud/internal/v2/gen/api"
)

// GetDeviceNotifications implements [api.Handler].
func (h *Handler) GetDeviceNotifications(
	ctx context.Context,
	params api.GetDeviceNotificationsParams,
) (*api.GetDeviceNotificationsOK, error) {
	panic("unimplemented")
}
