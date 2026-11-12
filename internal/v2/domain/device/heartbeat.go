package device

import "time"

type Heartbeat struct {
	Timestamp time.Time `validate:"required"`
}
