-- +goose Up
CREATE TABLE IF NOT EXISTS sensor_metrics (
    device_id String,
    sensor_id String,
    metric_name LowCardinality(String),
    metric_type LowCardinality(String),
    unit LowCardinality(String),
    value Float64,
    recorded_at DateTime64(3, 'UTC'),
    ingested_at DateTime64(3, 'UTC') DEFAULT now()
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(recorded_at)
ORDER BY (device_id, sensor_id, metric_name, recorded_at);

-- +goose Down
DROP TABLE IF EXISTS sensor_metrics;
