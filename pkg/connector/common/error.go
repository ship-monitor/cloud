package common

import (
	"context"

	"github.com/lerenn/asyncapi-codegen/pkg/extensions"
	"go.uber.org/zap"
)

func ErrorHandler(logger *zap.Logger) extensions.ErrorHandler {
	logger = logger.Named("ErrorHandler")

	return func(ctx context.Context, topic string, msg *extensions.AcknowledgeableBrokerMessage, err error) {
		logger.Error(
			"Error while handling message",
			zap.Error(err),
			zap.String("topic", topic),
		)
	}
}
