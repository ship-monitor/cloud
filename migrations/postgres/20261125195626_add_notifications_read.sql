-- +goose Up
CREATE TABLE user_notification_states (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_id UUID NOT NULL
        REFERENCES device_notifications(id) ON DELETE CASCADE,
    is_read BOOLEAN NOT NULL,

    PRIMARY KEY (user_id, notification_id)
);

-- +goose Down
DROP TABLE user_notification_states;
