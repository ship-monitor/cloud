// Package metrics defines domain types for device and sensor metrics.
package metrics

import "time"

// Type is the kind of a metric sample.
type Type string

const (
	TypeGauge     Type = "GAUGE"
	TypeCounter   Type = "COUNTER"
	TypeHistogram Type = "HISTOGRAM"
)

// Metric is a single metric sample captured at Timestamp (UTC).
type Metric struct {
	Name      string
	Type      Type
	Unit      string
	Value     float64
	Timestamp time.Time
}

// Query constrains metric queries over the metrics tables.
type Query struct {
	// From is an optional lower bound (inclusive) on recorded_at.
	From *time.Time
	// To is an optional upper bound (inclusive) on recorded_at.
	To *time.Time
	// Names optionally restricts the query to these metric names.
	Names []string
	// Limit caps the number of returned samples; zero means no limit.
	Limit uint64
}
