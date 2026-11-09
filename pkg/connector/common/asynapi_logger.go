package common

import (
	"context"

	"github.com/lerenn/asyncapi-codegen/pkg/extensions"
	"go.uber.org/zap"
)

type queueLogger struct {
	zap *zap.Logger
}

func LoggerFromZap(logger *zap.Logger) extensions.Logger {
	return &queueLogger{
		zap: logger,
	}
}

// Error implements [extensions.Logger].
func (q *queueLogger) Error(
	ctx context.Context,
	msg string,
	info ...extensions.LogInfo,
) {
	q.zap.Error(msg, zap.Reflect("values", info))
}

// Info implements [extensions.Logger].
func (q *queueLogger) Info(
	ctx context.Context,
	msg string,
	info ...extensions.LogInfo,
) {
	q.zap.Info(msg, zap.Reflect("values", info))
}

// Warning implements [extensions.Logger].
func (q *queueLogger) Warning(
	ctx context.Context,
	msg string,
	info ...extensions.LogInfo,
) {
	q.zap.Warn(msg, zap.Reflect("values", info))
}

var _ extensions.Logger = (*queueLogger)(nil)
