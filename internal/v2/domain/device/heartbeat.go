package device

import "time"

type Heartbeat struct {
	Timestamp time.Time `json:"timestamp" validate:"required"`
}
