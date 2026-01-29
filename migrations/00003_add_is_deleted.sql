-- +goose Up
ALTER TABLE urls ADD COLUMN is_deleted BOOLEAN DEFAULT FALSE NOT NULL;

-- +goose Down
ALTER TABLE urls DROP COLUMN is_deleted;
