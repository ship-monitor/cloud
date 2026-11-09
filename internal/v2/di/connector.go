package di

import (
	"context"
	"fmt"

	"github.com/ship-monitor/cloud/internal/v2/adapters/subscriber"
	"github.com/ship-monitor/cloud/internal/v2/handlers/queue"
	queuesecurity "github.com/ship-monitor/cloud/internal/v2/handlers/queue_security"
	"github.com/ship-monitor/cloud/pkg/connector/servers"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func provideConnector() fx.Option {
	return fx.Module("connector",
		fx.Provide(newDevServer),
		fx.Provide(newSubscriber),
	)
}

func newDevServer(lc fx.Lifecycle) (*servers.Development, error) {
	url, err := servers.DevelopmentURL()
	if err != nil {
		return nil, fmt.Errorf("dev url: %w", err)
	}

	srv, err := servers.ConnectDevelopmentBidi(context.TODO(), url)
	if err != nil {
		return nil, fmt.Errorf("connect dev server: %w", err)
	}

	lc.Append(fx.StopHook(srv.Close))

	return &srv.Development, nil
}

func newSubscriber(
	lc fx.Lifecycle,
	conn *servers.Development,
	logger *zap.Logger,
	handler *queue.Handler,
	security *queuesecurity.Handler,
) *subscriber.S {
	subs := subscriber.New(conn, logger, handler, security)

	lc.Append(fx.StartStopHook(subs.Start, func() error {
		err := subs.Stop()
		if err != nil {
			return fmt.Errorf("stop subscriber: %w", err)
		}

		return nil
	}))

	return subs
}
