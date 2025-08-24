-- +goose Up
ALTER TABLE locations ADD COLUMN timezone TEXT;

-- +goose Down
ALTER TABLE locations DROP COLUMN timezone;
