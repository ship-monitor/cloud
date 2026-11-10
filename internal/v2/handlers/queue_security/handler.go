package queuesecurity

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/adapters/subscriber"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/pkg/connector/asyncapi"
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

// HandleMessage implements [subscriber.SecurityHandler].
func (h *Handler) HandleMessage(
	ctx context.Context,
	header asyncapi.IdentificationHeadersSchema,
) (context.Context, error) {
	h.logger.Debug("Message header", zap.Any("header", header))

	deviceID, err := uuid.Parse(header.Authorization)
	if err != nil {
		return ctx, fmt.Errorf("parse uuid: %w", err)
	}

	return device.WithDeviceID(ctx, deviceID), nil
}
