package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"charm.land/log/v2"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/go-playground/validator/v10"
	"github.com/ship-monitor/cloud/internal/domain"
	"github.com/ship-monitor/cloud/pkg"
	"github.com/spf13/viper"
)

const deviceStatesTable = "device_states"

var _ pkg.MigrationRepo = (*DeviceStatesRepository)(nil)

// DeviceStatesRepository manages device state history in ClickHouse.
type DeviceStatesRepository struct {
	config *viper.Viper
	logger *log.Logger
	ch     clickhouse.Conn
}

func NewDeviceStatesRepo(
	ch clickhouse.Conn,
	config *viper.Viper,
	logger *log.Logger,
) *DeviceStatesRepository {
	return &DeviceStatesRepository{
		config: config,
		logger: logger.WithPrefix("Device States Repository"),
		ch:     ch,
	}
}

// Migrate implements [pkg.MigrationRepo].
func (c *DeviceStatesRepository) Migrate(ctx context.Context) error {
	const query = `
		CREATE TABLE IF NOT EXISTS device_states
		(
			device_id String,
			state String,
			timestamp DateTime64(6, 'UTC'),
			value String,
			inserted_at DateTime64(6, 'UTC') DEFAULT now64(6)
		)
		ENGINE = MergeTree
		ORDER BY (device_id, state, timestamp)
	`

	if err := c.ch.Exec(ctx, query); err != nil {
		return fmt.Errorf("create %s table: %w", deviceStatesTable, err)
	}

	return nil
}

func (c *DeviceStatesRepository) AddRecord(
	ctx context.Context,
	record domain.StateRecord,
) error {
	if err := validator.New().Struct(record); err != nil {
		return fmt.Errorf("invalid state record: %w", err)
	}

	value, err := json.Marshal(record.Value)
	if err != nil {
		return fmt.Errorf("marshal state value: %w", err)
	}

	const query = `
		INSERT INTO device_states (device_id, state, timestamp, value)
		VALUES (?, ?, ?, ?)
	`

	if err := c.ch.Exec(
		ctx,
		query,
		record.DeviceID,
		record.State,
		record.Timestamp.UTC(),
		string(value),
	); err != nil {
		return fmt.Errorf(
			"insert state record for device %q and state %q: %w",
			record.DeviceID,
			record.State,
			err,
		)
	}

	return nil
}

func (c *DeviceStatesRepository) GetStates(
	ctx context.Context,
	deviceID, state string,
	historyLength int,
) ([]domain.StateRecord, error) {
	if historyLength == 0 {
		historyLength = c.maxHistoryLength()
	}

	if historyLength <= 0 {
		return []domain.StateRecord{}, nil
	}

	const query = `
		SELECT timestamp, value
		FROM device_states
		WHERE device_id = ? AND state = ?
		ORDER BY timestamp DESC, inserted_at DESC
		LIMIT ?
	`

	rows, err := c.ch.Query(ctx, query, deviceID, state, historyLength)
	if err != nil {
		return nil, fmt.Errorf(
			"query state records for device %q and state %q: %w",
			deviceID,
			state,
			err,
		)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			c.logger.Warn("Failed to close state query", "error", err)
		}
	}()

	records := make([]domain.StateRecord, 0, historyLength)

	for rows.Next() {
		var (
			record domain.StateRecord
			value  string
		)

		if err := rows.Scan(&record.Timestamp, &value); err != nil {
			return nil, fmt.Errorf("scan state record: %w", err)
		}

		if err := json.Unmarshal([]byte(value), &record.Value); err != nil {
			return nil, fmt.Errorf("unmarshal state value: %w", err)
		}

		record.DeviceID = deviceID
		record.State = state
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate state records: %w", err)
	}

	return records, nil
}

func (c *DeviceStatesRepository) maxHistoryLength() int {
	return c.config.GetInt("states.max-history-length")
}
