package common

import (
	"context"

	"github.com/lerenn/asyncapi-codegen/pkg/extensions"
	"go.uber.org/zap"
)

var _ extensions.Logger = (*Logger)(nil)

type Logger struct {
	zap *zap.Logger
}

func LoggerFromZap(logger *zap.Logger) *Logger {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Logger{
		zap: logger.WithOptions(zap.WithCaller(false)),
	}
}

// Error implements [extensions.Logger].
func (q *Logger) Error(
	_ context.Context,
	msg string,
	info ...extensions.LogInfo,
) {
	q.zap.Error(msg, logInfoToFields(info)...)
}

// Info implements [extensions.Logger].
func (q *Logger) Info(
	_ context.Context,
	msg string,
	info ...extensions.LogInfo,
) {
	q.zap.Info(msg, logInfoToFields(info)...)
}

// Warning implements [extensions.Logger].
func (q *Logger) Warning(
	_ context.Context,
	msg string,
	info ...extensions.LogInfo,
) {
	q.zap.Warn(msg, logInfoToFields(info)...)
}

func logInfoToFields(info []extensions.LogInfo) []zap.Field {
	fields := make([]zap.Field, 0, len(info))
	for _, i := range info {
		fields = append(fields, zap.Any(i.Key, i.Value))
	}

	return fields
}
