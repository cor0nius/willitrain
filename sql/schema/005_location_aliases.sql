-- +goose Up
CREATE TABLE location_aliases (
    alias TEXT PRIMARY KEY,
    location_id UUID REFERENCES locations(id) ON DELETE CASCADE NOT NULL
);

-- +goose Down
DROP TABLE location_aliases;