package services

import (
	"context"
	"encoding/json"
	"fmt"

	"charm.land/log/v2"
	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
)

type DeviceStateCacheConfig struct {
	MaxHistoryLength int64 `validate:"required,gt=0"`
}

type DeviceStatesCache struct {
	rdb    *redis.Client
	config DeviceStateCacheConfig
	logger *log.Logger
}

func NewDeviceStatesCache(
	rdb *redis.Client,
	config DeviceStateCacheConfig,
	logger *log.Logger,
) *DeviceStatesCache {
	return &DeviceStatesCache{rdb: rdb, config: config, logger: logger.WithPrefix("state_cache")}
}

func (c *DeviceStatesCache) AddRecord(ctx context.Context, record domain.StateRecord) error {
	if err := validator.New().Struct(record); err != nil {
		return fmt.Errorf("invalid state record: %w", err)
	}

	key := getStateKey(record.DeviceID, record.State)

	val, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	length, err := c.rdb.LPush(ctx, key, val).Result()
	if err != nil {
		return fmt.Errorf("add record for %s: %w", key, err)
	}

	if length > c.config.MaxHistoryLength {
		go func() {
			if err := c.TrimRecords(ctx, record.DeviceID, record.State); err != nil {
				c.logger.Warn("Failed to trim records", "error", err)
			}
		}()
	}

	return nil
}

func (c *DeviceStatesCache) TrimRecords(ctx context.Context, deviceID, state string) error {
	key := getStateKey(deviceID, state)
	if err := c.rdb.LTrim(ctx, key, 0, c.config.MaxHistoryLength).Err(); err != nil {
		return fmt.Errorf("failed to trim records for %s: %w", key, err)
	}

	return nil
}

func getStateKey(deviceID, state string) string {
	return fmt.Sprintf("%s-state-%s", deviceID, state)
}
