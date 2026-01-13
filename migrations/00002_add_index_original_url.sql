-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_original_url ON urls(original_url);

-- +goose Down
DROP INDEX IF EXISTS idx_unique_original_url;
