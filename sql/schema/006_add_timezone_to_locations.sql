-- +goose Up
-- Adds a timezone column to the locations table to store the IANA Time Zone identifier (e.g., "Europe/Warsaw").
ALTER TABLE locations ADD COLUMN timezone TEXT;

-- +goose Down
ALTER TABLE locations DROP COLUMN timezone;
