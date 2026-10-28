-- +goose Up

CREATE TABLE users (
    id varchar primary key,
    name varchar not null,
    email varchar unique not null,
    password_hash varchar not null
);

-- +goose Down
DROP TABLE users;
