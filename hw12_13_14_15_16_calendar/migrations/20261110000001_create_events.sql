-- +goose Up
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE events (
    id            uuid        PRIMARY KEY,
    title         text        NOT NULL,
    start_at      timestamptz NOT NULL,
    end_at        timestamptz NOT NULL,
    description   text        NOT NULL DEFAULT '',
    user_id       bigint      NOT NULL,
    notify_before bigint      NOT NULL DEFAULT 0,
    CONSTRAINT events_time_check CHECK (end_at > start_at),
    CONSTRAINT events_no_overlap EXCLUDE USING gist (
        user_id WITH =,
        tstzrange(start_at, end_at) WITH &&
    )
);

CREATE INDEX events_start_at_idx ON events (start_at);

-- +goose Down
DROP TABLE events;
