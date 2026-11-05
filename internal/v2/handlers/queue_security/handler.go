package queuesecurity

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	nats "github.com/nats-io/nats.go"
	"github.com/ship-monitor/cloud/internal/v2/adapters/subscriber"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/pkg/connector/schemas/messages"
	"github.com/ship-monitor/cloud/pkg/headers"
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

// HandleMessage implements [subscriber.SecurityHandler].
func (h *Handler) HandleMessage(
	ctx context.Context,
	msg *nats.Msg,
) (context.Context, error) {
	var id messages.IdentificationHeader

	err := headers.NewDecoder(&msg.Header).Decode(&id)
	if err != nil {
		h.logger.Error(
			"Decode headers",
			zap.Error(err),
			zap.Any("headers", msg.Header),
		)

		return ctx, fmt.Errorf("decode headers: %w", err)
	}

	if id.DeviceID == uuid.Nil {
		return ctx, errNoID
	}

	return device.WithDeviceID(ctx, id.DeviceID), nil
}
