package logger

import (
	"os"

	"charm.land/log/v2"
)

func NewLogger() *log.Logger {
	logger := log.Default()

	if os.Getenv("DEVEL") != "" {
		logger.SetLevel(log.DebugLevel)
	}

	return logger
}
