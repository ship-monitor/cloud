package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/adapters/subscriber"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/ship-monitor/cloud/pkg/connector/asyncapi"
	"github.com/ship-monitor/cloud/pkg/hub"
	"go.uber.org/zap"
)

var _ subscriber.Handler = (*Handler)(nil)

type AliveRepo interface {
	InsertHeartbeat(
		ctx context.Context,
		deviceID uuid.UUID,
		h hub.Heartbeat,
	) error
}

type DeviceService interface {
	ApproveConnection(
		ctx context.Context,
		requestID uuid.UUID,
	) error
}

type Handler struct {
	repo   AliveRepo
	device DeviceService
	logger *zap.Logger
}

func New(
	repo AliveRepo,
	device DeviceService,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		repo:   repo,
		device: device,
		logger: logger,
	}
}

var errNoID = errors.New("no device id found in context")

// HandleHeartbeat implements [subscriber.Handler].
func (h *Handler) HandleHeartbeat(
	ctx context.Context,
	msg asyncapi.HeartbeatMessage,
) error {
	id, ok := device.DeviceIDFromContext(ctx)
	if !ok {
		return errNoID
	}

	err := h.repo.InsertHeartbeat(ctx, id, hub.Heartbeat{
		Timestamp: time.Unix(msg.Payload.Timestamp, 0),
	})
	if err != nil {
		return fmt.Errorf("insert heartbeat: %w", err)
	}

	return nil
}

func (h *Handler) HandleConnectResponse(
	ctx context.Context,
	response *asyncapi.ConnectResponseMessage,
) error {
	requestID, err := uuid.Parse(response.Payload.RequestId)
	if err != nil {
		return fmt.Errorf("parse request id: %w", err)
	}

	err = h.device.ApproveConnection(ctx, requestID)
	if err != nil {
		return fmt.Errorf("approve connection: %w", err)
	}

	return nil
}
