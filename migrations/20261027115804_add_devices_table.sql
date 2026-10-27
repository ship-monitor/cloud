-- +goose Up
CREATE TABLE devices (
    id varchar primary key,
    owner_id varchar,
    name varchar
);

-- +goose Down
DROP TABLE devices;
