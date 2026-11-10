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
	_ context.Context,
	msg string,
	info ...extensions.LogInfo,
) {
	q.zap.Error(msg, logInfoToFields(info)...)
}

// Info implements [extensions.Logger].
func (q *queueLogger) Info(
	_ context.Context,
	msg string,
	info ...extensions.LogInfo,
) {
	q.zap.Info(msg, logInfoToFields(info)...)
}

// Warning implements [extensions.Logger].
func (q *queueLogger) Warning(
	_ context.Context,
	msg string,
	info ...extensions.LogInfo,
) {
	q.zap.Warn(msg, logInfoToFields(info)...)
}

var _ extensions.Logger = (*queueLogger)(nil)

func logInfoToFields(info []extensions.LogInfo) []zap.Field {
	fields := make([]zap.Field, 0, len(info))
	for _, i := range info {
		fields = append(fields, zap.Any(i.Key, i.Value))
	}

	return fields
}
