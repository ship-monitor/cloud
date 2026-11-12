-- +goose Up

CREATE TABLE users (
    id varchar primary key,
    name varchar not null,
    email varchar unique not null,
    password_hash varchar not null,
    created_at TIMESTAMP WITH TIME ZONE not null,
    updated_at TIMESTAMP WITH TIME ZONE default null
);

-- +goose Down
DROP TABLE users;
