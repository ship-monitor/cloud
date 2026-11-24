-- +goose Up
CREATE TABLE devices (
    id UUID primary key,
    owner_id varchar not null,
    name TEXT not null,
    created_at TIMESTAMPTZ not null,
    updated_at TIMESTAMPTZ default null,
    manufacturer TEXT NOT NULL,
    model TEXT NOT NULL,
    revision TEXT NOT NULL,
    software TEXT NOT NULL
);

-- +goose Down
DROP TABLE devices;
