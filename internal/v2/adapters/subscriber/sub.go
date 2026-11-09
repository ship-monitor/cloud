package subscriber

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/ship-monitor/cloud/pkg/connector/channels"
	"github.com/ship-monitor/cloud/pkg/connector/messages"
	"github.com/ship-monitor/cloud/pkg/connector/schemas"
	"github.com/ship-monitor/cloud/pkg/connector/servers"
	"go.uber.org/zap"
)

type Handler interface {
	HandleHeartbeat(ctx context.Context, msg messages.HeartbeatReceiver) error
	HandleConnectResponse(ctx context.Context, msg *nats.Msg) error
}

type SecurityHandler interface {
	HandleMessage(
		ctx context.Context,
		header schemas.IdentificationHeaders,
	) (context.Context, error)
}

type HeartbeatChannel interface {
	SubscribeHeartbeat(
		ctx context.Context,
		cb func(message messages.HeartbeatReceiver),
	) (err error)
}
type ConnectResponseChannel interface {
	SubscribeConnectResponse(
		ctx context.Context,
		cb func(message messages.ConnectResponseReceiver),
	) (err error)
}

type S struct {
	conn     *servers.Development
	logger   *zap.Logger
	handler  Handler
	security SecurityHandler

	cancel context.CancelFunc
}

func New(
	conn *servers.Development,
	logger *zap.Logger,
	handler Handler,
	security SecurityHandler,
) *S {
	return &S{
		conn:     conn,
		logger:   logger.Named("Subscriber"),
		handler:  handler,
		security: security,
		cancel:   nil,
	}
}

func (s *S) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	deviceAlive, err := s.conn.OpenDevicesAliveNats(
		ctx,
		channels.DevicesAliveParameters{
			DeviceID: "*",
		}, nil,
	)
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}

	connectResponse, err := s.conn.OpenConnectResponsesNats(
		ctx,
		channels.ConnectResponsesParameters{
			DeviceID: "*",
		}, nil,
	)
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}

	go func() {
		err = deviceAlive.SubscribeHeartbeat(ctx, s.HandleHeartbeat)
		if err != nil {
			s.logger.Error("Subscribe", zap.Error(err))
		}
	}()
	go func() {
		err := connectResponse.SubscribeConnectResponse(
			ctx,
			s.HandleConnectResponse,
		)
		if err != nil {
			s.logger.Error("Subscribe", zap.Error(err))
		}
	}()

	return nil
}

func (s *S) Stop() error {
	s.cancel()

	return nil
}

func (s *S) HandleHeartbeat(message messages.HeartbeatReceiver) {
	ctx := context.Background()

	ctx, err := s.security.HandleMessage(ctx, message.Headers())
	if err != nil {
		s.logger.Error("Security handle", zap.Error(err))

		return
	}

	err = s.handler.HandleHeartbeat(ctx, message)
	if err != nil {
		s.logger.Error("Handle heartbeat", zap.Error(err))

		return
	}
}
func (s *S) HandleConnectResponse(message messages.ConnectResponseReceiver) {}
