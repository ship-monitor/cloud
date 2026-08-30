package devices

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ship-monitor/cloud/internal/domain"
	"github.com/ship-monitor/cloud/pkg"
)

const deviceStatesTable = "device_states"

const MaxHistoryLength = 100

var _ pkg.MigrationRepo = (*Repository)(nil)

type Record struct {
	DeviceID   string
	State      string
	Timestamp  time.Time
	InsertedAt time.Time

	ValueStr   sql.NullString
	ValueFloat sql.NullFloat64
	ValueBool  sql.NullBool
	ValueInt   sql.NullInt64
}

func newRecord(record domain.StateRecord) Record {
	result := Record{ //nolint:exhaustruct_v5
		DeviceID:  record.DeviceID,
		State:     record.State,
		Timestamp: record.Timestamp,
	}
	if value, ok := record.Value.(string); ok {
		result.ValueStr = sql.NullString{
			Valid:  true,
			String: value,
		}
	} else if value, ok := record.Value.(int64); ok {
		result.ValueInt = sql.NullInt64{
			Valid: true,
			Int64: value,
		}
	} else if value, ok := record.Value.(float64); ok {
		result.ValueFloat = sql.NullFloat64{
			Valid:   true,
			Float64: value,
		}
	} else if value, ok := record.Value.(bool); ok {
		result.ValueBool = sql.NullBool{
			Valid: true,
			Bool:  value,
		}
	}

	return result
}

func toDomain(in []Record) []domain.StateRecord {
	records := make([]domain.StateRecord, 0, len(in))

	for _, dbRecord := range in {
		result := domain.StateRecord{ //nolint:exhaustruct_v5
			State:     dbRecord.State,
			Timestamp: dbRecord.Timestamp,
			DeviceID:  dbRecord.DeviceID,
		}

		switch {
		case dbRecord.ValueStr.Valid:
			result.Value = dbRecord.ValueStr.String
		case dbRecord.ValueInt.Valid:
			result.Value = dbRecord.ValueInt.Int64
		case dbRecord.ValueFloat.Valid:
			result.Value = dbRecord.ValueFloat.Float64
		case dbRecord.ValueBool.Valid:
			result.Value = dbRecord.ValueBool.Bool

		default:
			result.Value = 0
		}
	}

	return records
}

func (r *Repository) AddRecord(
	ctx context.Context,
	record domain.StateRecord,
) error {
	if err := validator.New().Struct(record); err != nil {
		return fmt.Errorf("invalid state record: %w", err)
	}

	dbRecord := newRecord(record)

	const query = `
		INSERT INTO device_states (device_id, state, timestamp, value_string, value_int, value_float, value_bool)
		VALUES (?, ?, ?, ?)
	`

	if err := r.ch.Exec(
		ctx,
		query,
		record.DeviceID,
		record.State,
		record.Timestamp.UTC(),
		dbRecord.ValueStr,
		dbRecord.ValueInt,
		dbRecord.ValueFloat,
		dbRecord.ValueBool,
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

func (r *Repository) GetStates(
	ctx context.Context,
	deviceID, state string,
	historyLength int,
) ([]domain.StateRecord, error) {
	if historyLength == 0 {
		historyLength = MaxHistoryLength
	}

	if historyLength <= 0 {
		return []domain.StateRecord{}, nil
	}

	const query = `
		SELECT timestamp, value_string, value_int, value_float, value_bool
		FROM device_states
		WHERE device_id = ? AND state = ?
		ORDER BY timestamp DESC, inserted_at DESC
		LIMIT ?
	`

	rows, err := r.ch.Query(ctx, query, deviceID, state, historyLength)
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
			r.logger.Warn("Failed to close state query", "error", err)
		}
	}()

	records := make([]Record, 0, historyLength)

	for rows.Next() {
		var record Record

		if err := rows.Scan(
			&record.Timestamp,
			&record.ValueStr,
			&record.ValueInt,
			&record.ValueFloat,
			&record.ValueBool,
		); err != nil {
			return nil, fmt.Errorf("scan state record: %w", err)
		}

		record.DeviceID = deviceID
		record.State = state
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate state records: %w", err)
	}

	return toDomain(records), nil
}
