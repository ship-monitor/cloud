package alive

import (
	"time"

	"github.com/ship-monitor/cloud/pkg/hub"
)

type HeartbeatRecord struct {
	hub.Heartbeat

	CreatedAt time.Time `json:"createdAt"`
}

func NewRecord(h hub.Heartbeat) HeartbeatRecord {
	return HeartbeatRecord{
		Heartbeat: h,
		CreatedAt: time.Now().UTC(),
	}
}
