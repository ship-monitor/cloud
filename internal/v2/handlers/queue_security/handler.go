package queuesecurity

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	nats "github.com/nats-io/nats.go"
	"github.com/ship-monitor/cloud/internal/v2/adapters/subscriber"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
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
		logger: logger,
	}
}

var errNoID = errors.New("no device id provided in header")

const DeviceIDHeader = "device-id"

// HandleMessage implements [subscriber.SecurityHandler].
func (h *Handler) HandleMessage(
	ctx context.Context,
	msg *nats.Msg,
) (context.Context, error) {
	id := msg.Header.Get(DeviceIDHeader)

	deviceID, err := uuid.Parse(id)
	if err != nil {
		return ctx, fmt.Errorf("parse device is: %w", err)
	}

	if deviceID == uuid.Nil {
		return ctx, errNoID
	}

	return device.WithDeviceID(ctx, deviceID), nil
}
