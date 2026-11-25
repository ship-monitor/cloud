-- +goose Up
ALTER TABLE user_notification_states
    ADD CONSTRAINT user_notification_states_user_notification_key
    UNIQUE (user_id, notification_id);

-- +goose Down
ALTER TABLE user_notification_states
    DROP CONSTRAINT user_notification_states_user_notification_key;
