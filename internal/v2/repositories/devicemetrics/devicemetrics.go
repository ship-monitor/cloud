// Package devicemetrics provides a ClickHouse-backed repository for the
// device_metrics table.
//
//nolint:varnamelen
package devicemetrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/v2/domain/metrics"
)

// Repository reads and writes device metrics in ClickHouse.
type Repository struct {
	conn driver.Conn
}

// New returns a Repository backed by the given ClickHouse connection.
func New(conn driver.Conn) *Repository {
	return &Repository{conn: conn}
}

// Query constrains device metric queries.
type Query = metrics.Query

// InsertBatch stores a batch of metric samples for the given device in
// one ClickHouse insert.
func (r *Repository) InsertBatch(
	ctx context.Context,
	deviceID uuid.UUID,
	ms []metrics.Metric,
) error {
	if len(ms) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx,
		`INSERT INTO device_metrics
		 (device_id, metric_name, metric_type, unit, value, recorded_at)`,
	)
	if err != nil {
		return fmt.Errorf("prepare device metrics batch: %w", err)
	}
	defer func() { _ = batch.Close() }()

	for _, m := range ms {
		err = batch.Append(
			deviceID.String(),
			m.Name,
			string(m.Type),
			m.Unit,
			m.Value,
			m.Timestamp.UTC(),
		)
		if err != nil {
			return fmt.Errorf("append device metric: %w", err)
		}
	}

	err = batch.Send()
	if err != nil {
		return fmt.Errorf("send device metrics batch: %w", err)
	}

	return nil
}

// List returns metric samples for the given device matching the filter,
// ordered by recorded_at.
func (r *Repository) List(
	ctx context.Context,
	deviceID uuid.UUID,
	filter Query,
) ([]metrics.Metric, error) {
	query := strings.Builder{}
	args := []any{}

	query.WriteString(
		`SELECT metric_name, metric_type, unit, value, recorded_at
		 FROM device_metrics WHERE device_id = ?`,
	)

	args = append(args, deviceID.String())

	if filter.From != nil {
		query.WriteString(" AND recorded_at >= ?")

		args = append(args, filter.From.UTC())
	}

	if filter.To != nil {
		query.WriteString(" AND recorded_at <= ?")

		args = append(args, filter.To.UTC())
	}

	if len(filter.Names) > 0 {
		placeholders := strings.TrimSuffix(
			strings.Repeat("?, ", len(filter.Names)), ", ",
		)
		query.WriteString(" AND metric_name IN (" + placeholders + ")")

		for _, name := range filter.Names {
			args = append(args, name)
		}
	}

	query.WriteString(" ORDER BY recorded_at")

	if filter.Limit > 0 {
		query.WriteString(" LIMIT ?")

		args = append(args, filter.Limit)
	}

	rows, err := r.conn.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("select device metrics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ms []metrics.Metric

	for rows.Next() {
		m, err := scanMetric(rows)
		if err != nil {
			return nil, fmt.Errorf("scan device metric: %w", err)
		}

		ms = append(ms, m)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("iterate device metrics: %w", err)
	}

	return ms, nil
}

func scanMetric(rows driver.Rows) (metrics.Metric, error) {
	var (
		m       metrics.Metric
		mType   string
		stamped time.Time
	)

	err := rows.Scan(&m.Name, &mType, &m.Unit, &m.Value, &stamped)
	if err != nil {
		return metrics.Metric{}, fmt.Errorf("scan device metric: %w", err)
	}

	m.Type = metrics.Type(mType)
	m.Timestamp = stamped

	return m, nil
}
