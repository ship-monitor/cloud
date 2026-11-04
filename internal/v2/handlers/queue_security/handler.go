package queuesecurity

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	nats "github.com/nats-io/nats.go"
	"github.com/ship-monitor/cloud/internal/v2/adapters/subscriber"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
)

var _ subscriber.SecurityHandler = (*Handler)(nil)

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

const header = "x-device-id"

// HandleMessage implements [subscriber.SecurityHandler].
func (h *Handler) HandleMessage(
	ctx context.Context,
	msg *nats.Msg,
) (context.Context, error) {
	id, err := uuid.Parse(msg.Header.Get(header))
	if err != nil {
		return ctx, fmt.Errorf("parse id from header %q: %w", header, err)
	}

	return device.WithDeviceID(ctx, id), nil
}
