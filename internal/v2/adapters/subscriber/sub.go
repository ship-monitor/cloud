package subscriber

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/ship-monitor/cloud/pkg/connector/channels"
	"go.uber.org/zap"
)

type Handler interface {
	HandleHeartbeat(ctx context.Context, msg *nats.Msg) error
	HandleConnectResponse(ctx context.Context, msg *nats.Msg) error
}

type SecurityHandler interface {
	HandleMessage(ctx context.Context, msg *nats.Msg) (context.Context, error)
}

type S struct {
	conn   *nats.Conn
	logger *zap.Logger

	handler  Handler
	security SecurityHandler

	subs *subsContainer
}

func New(
	conn *nats.Conn,
	logger *zap.Logger,
	handler Handler,
	security SecurityHandler,
) *S {
	return &S{
		conn:     conn,
		logger:   logger.Named("Subscriber"),
		handler:  handler,
		security: security,
		subs:     newSubsContainer(),
	}
}

func (s *S) Start() {
	s.subscribe(
		channels.DevicesAliveAddress(channels.DevicesAliveParameters{
			DeviceID: "*",
		}).String(),
		s.handler.HandleHeartbeat)
	s.subscribe(
		channels.ConnectResponsesAddress(channels.ConnectResponsesParameters{
			DeviceID: "*",
		}).String(),
		s.handler.HandleConnectResponse,
	)
}

func (s *S) Stop() error {
	err := s.subs.Unsubscribe()
	if err != nil {
		return fmt.Errorf("unsubscribe all subjects: %w", err)
	}

	return nil
}

func (s *S) subscribe(subj string, h HandlerFunc) {
	sub, err := s.conn.Subscribe(
		subj,
		s.wrapHandler(
			h,
		),
	)
	if err != nil {
		s.logger.Error(
			"Failed subscribe",
			zap.Error(err),
			zap.String("subject", subj),
		)
	}

	s.subs.Add(sub)
}

const HandleTimeout = time.Second * 10

type HandlerFunc func(ctx context.Context, msg *nats.Msg) error

func (s *S) wrapHandler(
	handler HandlerFunc,
) nats.MsgHandler {
	return func(msg *nats.Msg) {
		logger := s.logger.With(
			zap.String("subject", msg.Subject),
		)

		ctx, cancel := context.WithTimeout(context.Background(), HandleTimeout)
		defer cancel()

		ctx, err := s.security.HandleMessage(ctx, msg)
		if err != nil {
			logger.Error(
				"Security handle",
				zap.Error(err),
			)

			err = msg.Ack()
			if err != nil {
				logger.Error(
					"Acknowledge message",
					zap.Error(err),
				)
			}

			return
		}

		err = handler(ctx, msg)
		if err != nil {
			logger.Error(
				"Message handler",
				zap.Error(err),
			)

			return
		}

		logger.Info("Handled message")
	}
}
