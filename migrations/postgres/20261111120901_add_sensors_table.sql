-- +goose Up
CREATE TABLE sensors (
    id UUID not null,
    device_id UUID not null REFERENCES devices(id),
    name TEXT not null,
    display_name TEXT not null,
    created_at TIMESTAMPTZ not null,
    updated_at TIMESTAMPTZ default null,

    primary key (id, device_id)
);

-- +goose Down
DROP TABLE sensors;
