package alive

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/device"
	"github.com/valkey-io/valkey-go"
)

const ExpirationDuration = time.Hour * 48

type Repo struct {
	valkey valkey.Client
}

func New(
	valkey valkey.Client,
) *Repo {
	return &Repo{
		valkey: valkey,
	}
}

func (r *Repo) InsertHeartbeat(
	ctx context.Context,
	deviceID uuid.UUID,
	h device.Heartbeat,
) error {
	record := NewRecord(h)

	data, err := json.Marshal(&record)
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	err = r.valkey.Do(ctx,
		r.valkey.B().
			Setex().
			Key(key(deviceID)).
			Seconds(int64(ExpirationDuration.Seconds())).
			Value(string(data)).
			Build(),
	).Error()
	if err != nil {
		return fmt.Errorf("set value: %w", err)
	}

	return nil
}

func (r *Repo) GetLastHeartbeat(
	ctx context.Context,
	deviceID uuid.UUID,
) (*device.Heartbeat, error) {
	var record HeartbeatRecord

	err := r.valkey.Do(ctx,
		r.valkey.B().Get().Key(key(deviceID)).Build(),
	).DecodeJSON(&record)
	if err != nil {
		return nil, fmt.Errorf("get value: %w", err)
	}

	return &record.Heartbeat, nil
}

func key(deviceID uuid.UUID) string {
	return "device." + deviceID.String() + ".heartbeats"
}
