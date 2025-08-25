-- +goose Up
-- locations stores the geographic coordinates and other details for a given city.
CREATE TABLE locations (
    id UUID PRIMARY KEY,
    city_name TEXT UNIQUE NOT NULL,
    latitude FLOAT NOT NULL,
    longitude FLOAT NOT NULL,
    country_code TEXT NOT NULL
);

-- +goose Down
DROP TABLE locations;