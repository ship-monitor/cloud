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
	queueSecurity "github.com/ship-monitor/cloud/internal/v2/handlers/queue_security"
	"github.com/ship-monitor/cloud/internal/v2/services/device"
	metricsService "github.com/ship-monitor/cloud/internal/v2/services/metrics"
	"github.com/ship-monitor/cloud/internal/v2/services/notification"
	"github.com/ship-monitor/cloud/internal/v2/services/sensor"
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
	security *queueSecurity.Handler,
	sensor *sensor.Service,
	metrics *metricsService.Service,
	device *device.Service,
	notify *notification.Service,
) *subscriber.S {
	subs := subscriber.New(
		controller,
		logger,
		handler,
		security,
		sensor,
		metrics,
		device,
		notify,
	)

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

	logger = logger.Named("ConnectionController")

	qLog := common.LoggerFromZap(logger)

	controller, err := asyncapi.NewAppController(
		natsController,
		asyncapi.WithLogger(qLog),
		asyncapi.WithErrorHandler(common.ErrorHandler(logger)),
		asyncapi.WithMiddlewares(
			middlewares.Logging(qLog),
			middlewares.Recovery(qLog),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("new app controller: %w", err)
	}

	lc.Append(fx.StopHook(controller.Close))

	return controller, nil
}

func newPublisher(
	controller *asyncapi.AppController,
) *publisher.P {
	return publisher.New(controller)
}
