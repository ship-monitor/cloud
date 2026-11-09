package queuesecurity

import (
	"context"
	"errors"

	"github.com/ship-monitor/cloud/internal/v2/adapters/subscriber"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/pkg/connector/schemas"
	"go.uber.org/zap"
)

var _ subscriber.SecurityHandler = (*Handler)(nil)

type Handler struct {
	logger *zap.Logger
}

func New(
	logger *zap.Logger,
) *Handler {
	return &Handler{
		logger: logger.Named("QueueSecurityHandler"),
	}
}

var errNoID = errors.New("no device id provided in header")

const DeviceIDHeader = "DeviceID"

// HandleMessage implements [subscriber.SecurityHandler].
func (h *Handler) HandleMessage(
	ctx context.Context,
	header schemas.IdentificationHeaders,
) (context.Context, error) {
	h.logger.Debug("Message header", zap.Any("header", header))

	deviceID := header.DeviceID
	if deviceID == nil {
		return ctx, errNoID
	}

	return device.WithDeviceID(ctx, *deviceID), nil
}
