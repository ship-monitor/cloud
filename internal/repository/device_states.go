package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"charm.land/log/v2"
	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
)

// DeviceStatesRepository manages state records in redis.
type DeviceStatesRepository struct {
	rdb    *redis.Client
	config *viper.Viper
	logger *log.Logger
}

func NewDeviceStatesRepo(
	rdb *redis.Client,
	config *viper.Viper,
	logger *log.Logger,
) *DeviceStatesRepository {
	return &DeviceStatesRepository{
		rdb:    rdb,
		config: config,
		logger: logger.WithPrefix("Device States Repository"),
	}
}

func (c *DeviceStatesRepository) AddRecord(
	ctx context.Context,
	record domain.StateRecord,
) error {
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

	if length > c.maxHistoryLength() {
		trimCtx, cancel := context.WithTimeout(
			ctx,
			TrimRecordsTimeout,
		)
		defer cancel()

		go func(ctx context.Context) {
			if err := c.TrimRecords(
				ctx,
				record.DeviceID,
				record.State,
			); err != nil {
				c.logger.Warn("Failed to trim records", "error", err)
			}
		}(trimCtx)
	}

	return nil
}

const TrimRecordsTimeout = time.Second * 5

func (c *DeviceStatesRepository) TrimRecords(
	ctx context.Context,
	deviceID, state string,
) error {
	key := getStateKey(deviceID, state)
	if err := c.rdb.LTrim(ctx, key, 0, c.maxHistoryLength()).
		Err(); err != nil {
		return fmt.Errorf("failed to trim records for %s: %w", key, err)
	}

	return nil
}

func (c *DeviceStatesRepository) GetStates(
	ctx context.Context,
	deviceID, state string,
	historyLength int,
) ([]domain.StateRecord, error) {
	key := getStateKey(deviceID, state)

	if historyLength == 0 {
		historyLength = int(c.maxHistoryLength())
	}

	values, err := c.rdb.LRange(ctx, key, 0, int64(historyLength)).Result()
	if err != nil {
		return nil, fmt.Errorf("get record from redis: %w", err)
	}

	records := make([]domain.StateRecord, 0, len(values))

	for _, val := range values {
		var r domain.StateRecord

		err := json.Unmarshal([]byte(val), &r)
		if err != nil {
			log.Error("Failed unmarshal record", "error", err)

			continue
		}

		records = append(records, r)
	}

	return records, nil
}

func (c *DeviceStatesRepository) maxHistoryLength() int64 {
	return c.config.GetInt64("states.max-history-length")
}

func getStateKey(deviceID, state string) string {
	return fmt.Sprintf("%s-state-%s", deviceID, state)
}
