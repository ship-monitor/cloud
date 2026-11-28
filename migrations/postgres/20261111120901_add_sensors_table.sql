-- +goose Up
CREATE TABLE sensors (
    id UUID not null,
    device_id UUID not null REFERENCES devices(id) ON DELETE CASCADE,
    name TEXT not null,
    display_name TEXT not null,
    created_at TIMESTAMPTZ not null,
    updated_at TIMESTAMPTZ default null,
    manufacturer TEXT NOT NULL,
    model TEXT NOT NULL,
    revision TEXT NOT NULL,
    software TEXT NOT NULL,

    PRIMARY KEY (id, device_id)
);

-- +goose Down
DROP TABLE sensors;
