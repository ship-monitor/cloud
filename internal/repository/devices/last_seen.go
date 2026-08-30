package devices

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const LastSeenExpiration = time.Second * 30

func (r *Repository) Update(ctx context.Context, deviceID uuid.UUID) error {
	timestamp := time.Now()

	err := r.redis.Set(ctx, deviceLastSeenKey(deviceID), timestamp, LastSeenExpiration).
		Err()
	if err != nil {
		return fmt.Errorf("set value in redis by key: %w", err)
	}

	return nil
}

func (r *Repository) IsOnline(
	ctx context.Context,
	deviceID uuid.UUID,
) (bool, error) {
	now := time.Now()

	timestamp, err := r.redis.Get(ctx, deviceLastSeenKey(deviceID)).Time()
	if err != nil {
		return false, fmt.Errorf("get value from redis by key: %w", err)
	}

	isOnline := timestamp.Add(LastSeenExpiration).After(now)

	return isOnline, nil
}

func deviceLastSeenKey(id uuid.UUID) string {
	return "device." + id.String() + ".last-seen"
}
