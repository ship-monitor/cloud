-- +goose Up
CREATE TABLE IF NOT EXISTS device_metrics (
    device_id String,
    metric_name LowCardinality(String),
    metric_type LowCardinality(String),
    unit LowCardinality(String),
    value Float64,
    recorded_at DateTime64(3, 'UTC'),
    ingested_at DateTime64(3, 'UTC') DEFAULT now()
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(recorded_at)
ORDER BY (device_id, metric_name, recorded_at);

-- +goose Down
DROP TABLE IF EXISTS device_metrics;
