-- +goose Up
CREATE TABLE IF NOT EXISTS urls (
    id TEXT PRIMARY KEY,
    short_url TEXT NOT NULL UNIQUE,
    original_url TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS urls;
