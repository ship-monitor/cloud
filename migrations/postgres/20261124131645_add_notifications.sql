-- +goose Up
CREATE TABLE device_notifications (
    id UUID PRIMARY KEY,

    device_id UUID NOT NULL,
    sensor_id UUID,

    type TEXT NOT NULL,

    severity TEXT NOT NULL
        CHECK (severity IN ('info', 'warning', 'critical')),

    occurred_at TIMESTAMPTZ NOT NULL,
    reported_at TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    data JSONB NOT NULL DEFAULT '{}',

    CONSTRAINT notification_data_object
        CHECK (jsonb_typeof(data) = 'object')

);

CREATE INDEX idx_device_notifications_hub_time
    ON device_notifications (device_id, occurred_at DESC);

CREATE INDEX idx_device_notifications_sensor_time
    ON device_notifications (sensor_id, occurred_at DESC)
    WHERE sensor_id IS NOT NULL;

CREATE INDEX idx_device_notifications_severity
    ON device_notifications (severity, occurred_at DESC);

-- +goose Down
DROP TABLE device_notifications;
