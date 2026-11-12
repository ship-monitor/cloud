-- +goose Up
CREATE TABLE connect_requests (
    id varchar primary key,
    device_id varchar not null,
    user_id varchar not null,
    approved BOOLEAN not null,
    created_at TIMESTAMP WITH TIME ZONE not null,
    approved_at TIMESTAMP WITH TIME ZONE default null
);

-- +goose Down
DROP TABLE connect_requests;
