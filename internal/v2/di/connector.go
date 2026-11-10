package di

import (
	"context"
	"fmt"

	"github.com/lerenn/asyncapi-codegen/pkg/extensions/brokers/nats"
	"github.com/lerenn/asyncapi-codegen/pkg/extensions/middlewares"
	"github.com/ship-monitor/cloud/internal/v2/adapters/publisher"
	"github.com/ship-monitor/cloud/internal/v2/adapters/subscriber"
	"github.com/ship-monitor/cloud/internal/v2/config"
	"github.com/ship-monitor/cloud/internal/v2/handlers/queue"
	queuesecurity "github.com/ship-monitor/cloud/internal/v2/handlers/queue_security"
	"github.com/ship-monitor/cloud/pkg/connector/asyncapi"
	"github.com/ship-monitor/cloud/pkg/connector/common"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func provideConnector() fx.Option {
	return fx.Module("connector",
		fx.Provide(newSubscriber),
		fx.Provide(newConnectorController),

		fx.Invoke(func(*subscriber.S) {}),
	)
}

func newSubscriber(
	lc fx.Lifecycle,
	controller *asyncapi.AppController,
	logger *zap.Logger,
	handler *queue.Handler,
	security *queuesecurity.Handler,
) *subscriber.S {
	subs := subscriber.New(controller, logger, handler, security)

	lc.Append(fx.StartStopHook(subs.Start, func(ctx context.Context) error {
		err := subs.Stop(ctx)
		if err != nil {
			return fmt.Errorf("stop subscriber: %w", err)
		}

		return nil
	}))

	return subs
}

func newConnectorController(
	lc fx.Lifecycle,
	config *config.Config,
	logger *zap.Logger,
) (*asyncapi.AppController, error) {
	natsController, err := nats.NewController(config.NatsURL)
	if err != nil {
		return nil, fmt.Errorf("new nats controller: %w", err)
	}

	qLog := common.LoggerFromZap(logger.Named("ConnectorController"))

	c, err := asyncapi.NewAppController(
		natsController,
		asyncapi.WithLogger(qLog),
		asyncapi.WithMiddlewares(
			middlewares.Logging(qLog),
			middlewares.Recovery(qLog),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("new app controller: %w", err)
	}

	lc.Append(fx.StopHook(c.Close))

	return c, nil
}

func newPublisher(
	controller *asyncapi.AppController,
) *publisher.P {
	return publisher.New(controller)
}
