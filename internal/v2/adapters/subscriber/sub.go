package subscriber

import (
	"context"
	"errors"
	"fmt"

	"github.com/ship-monitor/cloud/pkg/connector/asyncapi"
	"go.uber.org/zap"
)

type SecurityHandler interface {
	HandleMessage(
		ctx context.Context,
		header asyncapi.IdentificationHeadersSchema,
	) (context.Context, error)
}

type Handler interface {
	HandleHeartbeat(
		ctx context.Context,
		msg asyncapi.HeartbeatMessage,
	) error
	HandleConnectResponse(
		ctx context.Context,
		response *asyncapi.ConnectResponseMessage,
	) error
}

// type HeartbeatChannel interface {
// 	SubscribeHeartbeat(
// 		ctx context.Context,
// 		cb func(message messages.HeartbeatReceiver),
// 	) (err error)
// }
// type ConnectResponseChannel interface {
// 	SubscribeConnectResponse(
// 		ctx context.Context,
// 		cb func(message messages.ConnectResponseReceiver),
// 	) (err error)
// }

type S struct {
	controller *asyncapi.AppController
	logger     *zap.Logger
	handler    Handler
	security   SecurityHandler

	cancel context.CancelFunc
}

func New(
	controller *asyncapi.AppController,
	logger *zap.Logger,
	handler Handler,
	security SecurityHandler,
) *S {
	return &S{
		controller: controller,
		logger:     logger.Named("Subscriber"),
		handler:    handler,
		security:   security,
		cancel:     nil,
	}
}

func (s *S) Start(ctx context.Context) error {
	err := errors.Join(

		s.controller.SubscribeToOnDeviceHeartbeatOperation(
			ctx,
			asyncapi.DevicesAliveChannelParameters{DeviceId: "*"},
			s.HandleHeartbeat,
		),
		s.controller.SubscribeToOnConnectResponseOperation(
			ctx,
			asyncapi.ConnectResponsesChannelParameters{DeviceId: "*"},
			s.HandleConnectResponse,
		),
	)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	return nil
}

func (s *S) Stop() error {
	s.cancel()

	return nil
}

func (s *S) HandleHeartbeat(
	ctx context.Context,
	message asyncapi.HeartbeatMessage,
) error {
	s.logger.Info("Handle heartbeat")

	ctx, err := s.security.HandleMessage(ctx, message.Headers)
	if err != nil {
		return fmt.Errorf("handle security: %w", err)
	}

	err = s.handler.HandleHeartbeat(ctx, message)
	if err != nil {
		return fmt.Errorf("handle heartbeat: %w", err)
	}

	return nil
}

func (s *S) HandleConnectResponse(
	ctx context.Context,
	message asyncapi.ConnectResponseMessage,
) error {
	s.logger.Info("Handle connect response")

	ctx, err := s.security.HandleMessage(ctx, message.Headers)
	if err != nil {
		return fmt.Errorf("handle security: %w", err)
	}

	err = s.handler.HandleConnectResponse(ctx, &message)
	if err != nil {
		return fmt.Errorf("handle connect response: %w", err)
	}

	return nil
}
