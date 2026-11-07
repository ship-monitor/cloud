-- +goose Up
CREATE TABLE connect_requests (
    id varchar primary key,
    device_id varchar not null,
    created_at DATETIME not null,
    user_id varchar not null,
    approved bit not null
);

-- +goose Down
DROP TABLE connect_requests;
