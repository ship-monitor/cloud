-- +goose Up

CREATE TABLE users (
    id varchar primary key,
    name varchar ,
    email varchar unique,
    password_hash varchar
);

-- +goose Down
DROP TABLE users;
