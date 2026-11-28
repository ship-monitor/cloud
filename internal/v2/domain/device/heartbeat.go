package device

import (
	"errors"
	"time"
)

type Heartbeat struct {
	Timestamp time.Time `json:"timestamp" validate:"required"`
}

var ErrNoHeartbeat = errors.New("no heartbeat was recorded yet")
