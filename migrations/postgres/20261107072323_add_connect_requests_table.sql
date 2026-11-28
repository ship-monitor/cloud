-- +goose Up
CREATE TABLE connect_requests (
    id UUID primary key,
    device_id UUID not null,
    user_id UUID not null REFERENCES users(id) ON DELETE CASCADE,
    approved BOOLEAN not null,
    created_at TIMESTAMPTZ not null,
    approved_at TIMESTAMPTZ default null
);

-- +goose Down
DROP TABLE connect_requests;
