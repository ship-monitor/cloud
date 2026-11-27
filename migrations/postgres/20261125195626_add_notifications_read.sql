-- +goose Up
CREATE TABLE user_notification_states (
    user_id UUID NOT NULL REFERENCES users(id),
    notification_id UUID NOT NULL REFERENCES device_notifications(id),
    is_read BOOLEAN NOT NULL
);

-- +goose Down
DROP TABLE user_notification_states;
