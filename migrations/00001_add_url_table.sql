-- +goose Up
CREATE TABLE IF NOT EXISTS urls (
    id VARCHAR(255) PRIMARY KEY,
    short_url VARCHAR(255) NOT NULL UNIQUE,
    original_url VARCHAR(2048) NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_original_url ON urls(original_url);

-- +goose Down
DROP INDEX IF EXISTS idx_unique_original_url;
DROP TABLE IF EXISTS urls;
