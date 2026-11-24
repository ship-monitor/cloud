-- +goose Up

CREATE TABLE users (
    id UUID primary key,
    name varchar not null,
    email varchar unique not null,
    password_hash varchar not null,
    created_at TIMESTAMPTZ not null,
    updated_at TIMESTAMPTZ default null
);

-- +goose Down
DROP TABLE users;
