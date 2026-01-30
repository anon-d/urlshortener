-- +goose Up
ALTER TABLE urls ADD COLUMN user_id VARCHAR(255);

-- +goose Down
ALTER TABLE urls DROP COLUMN user_id;
