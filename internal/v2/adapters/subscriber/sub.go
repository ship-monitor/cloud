package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/ship-monitor/cloud/pkg/hub"
	"go.uber.org/zap"
)

type Handler interface {
	HandleHeartbeat(ctx context.Context, h *hub.Heartbeat) error
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
		logger:   logger,
		handler:  handler,
		security: security,
		subs:     newSubsContainer(),
	}
}

func (s *S) Start() {
	s.subscribe(
		"device.*.health",
		wrapHandler(s.logger, s.security, s.handler.HandleHeartbeat),
	)
}

func (s *S) Stop() error {
	err := s.subs.Unsubscribe()
	if err != nil {
		return fmt.Errorf("unsubscribe all subjects: %w", err)
	}

	return nil
}

func (s *S) subscribe(subj string, handler nats.MsgHandler) {
	sub, err := s.conn.Subscribe(
		subj,
		handler,
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

func wrapHandler[T any](
	logger *zap.Logger,
	sec SecurityHandler,
	handler func(context.Context, *T) error,
) nats.MsgHandler {
	return func(msg *nats.Msg) {
		logger = logger.Named("Subject handler").With(
			zap.String("subject", msg.Subject),
		)

		ctx, cancel := context.WithTimeout(context.Background(), HandleTimeout)
		defer cancel()

		ctx, err := sec.HandleMessage(ctx, msg)
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

		var data T

		err = json.Unmarshal(msg.Data, &data)
		if err != nil {
			logger.Error(
				"Unmarshall message",
				zap.Error(err),
			)

			return
		}

		err = handler(ctx, &data)
		if err != nil {
			logger.Error(
				"Message handler",
				zap.Error(err),
			)

			return
		}

		err = msg.Ack()
		if err != nil {
			logger.Error(
				"Acknowledge message",
				zap.Error(err),
			)
		}
	}
}
