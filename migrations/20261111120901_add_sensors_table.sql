-- +goose Up
CREATE TABLE sensors (
    id varchar not null,
    device_id varchar not null,
    name varchar not null,
    display_name varchar not null,
    created_at TIMESTAMP WITH TIME ZONE not null,
    updated_at TIMESTAMP WITH TIME ZONE default null,

    primary key (id, device_id)
);

-- +goose Down
DROP TABLE sensors;
