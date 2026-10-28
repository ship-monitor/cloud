-- +goose Up
CREATE TABLE devices (
    id varchar primary key,
    owner_id varchar not null,
    name varchar not null,
    security_key varchar not null
);

-- +goose Down
DROP TABLE devices;
