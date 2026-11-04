package queue

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/adapters/subscriber"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/pkg/hub"
)

var _ subscriber.Handler = (*Handler)(nil)

type AliveRepo interface {
	InsertHertbeat(
		ctx context.Context,
		deviceID uuid.UUID,
		h hub.Heartbeat,
	) error
}

type Handler struct {
	repo AliveRepo
}

func New(
	repo AliveRepo,
) *Handler {
	return &Handler{
		repo: repo,
	}
}

var errNoID = errors.New("no device id found in context")

// HandleHeartbeat implements [subscriber.Handler].
func (h *Handler) HandleHeartbeat(
	ctx context.Context,
	beat *hub.Heartbeat,
) error {
	id, ok := device.DeviceIDFromContext(ctx)
	if !ok {
		return errNoID
	}

	err := h.repo.InsertHertbeat(ctx, id, *beat)
	if err != nil {
		return fmt.Errorf("insert heartbeat: %w", err)
	}

	return nil
}
