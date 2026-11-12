package alive

import (
	"time"

	"github.com/ship-monitor/cloud/internal/v2/domain/device"
)

type HeartbeatRecord struct {
	device.Heartbeat

	CreatedAt time.Time `json:"createdAt"`
}

func NewRecord(h device.Heartbeat) HeartbeatRecord {
	return HeartbeatRecord{
		Heartbeat: h,
		CreatedAt: time.Now().UTC(),
	}
}
