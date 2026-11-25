-- +goose Up
CREATE TABLE user_notification_states (
    user_id UUID NOT NULL,
    notification_id UUID NOT NULL,
    is_read BOOLEAN NOT NULL
);

-- +goose Down
DROP TABLE user_notification_states;
