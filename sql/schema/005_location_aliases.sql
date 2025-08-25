-- +goose Up
-- location_aliases maps alternative names (aliases) to a canonical location_id.
-- This allows users to search for locations using different names (e.g., "wroc" for "Wrocław").
CREATE TABLE location_aliases (
    alias TEXT PRIMARY KEY,
    location_id UUID REFERENCES locations(id) ON DELETE CASCADE NOT NULL
);

-- +goose Down
DROP TABLE location_aliases;