-- +goose Up
ALTER TABLE events ADD COLUMN notified_at timestamptz;

-- +goose Down
ALTER TABLE events DROP COLUMN notified_at;