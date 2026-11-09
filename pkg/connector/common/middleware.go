package common

import (
	"context"
	"fmt"

	"github.com/lerenn/asyncapi-codegen/pkg/extensions"
	"go.uber.org/zap"
)

func LoggingMiddleware(logger *zap.Logger) extensions.Middleware {
	return func(ctx context.Context, msg *extensions.BrokerMessage, next extensions.NextMiddleware) error {
		err := next(ctx)
		if err != nil {
			logger.Error(
				"Handle message",
				zap.Error(err),
				zap.String("message", msg.String()),
			)

			return fmt.Errorf("handle message: %w", err)
		}

		logger.Info("Handled message")

		return nil
	}
}
