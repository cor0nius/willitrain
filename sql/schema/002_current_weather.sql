-- +goose Up
CREATE TABLE current_weather (
    id UUID PRIMARY KEY,
    location_id UUID REFERENCES locations(id) ON DELETE CASCADE NOT NULL,
    source_api TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    temperature_c FLOAT,
    humidity INT,
    wind_speed_kmh FLOAT,
    precipitation_mm FLOAT,
    condition_text TEXT
);

-- +goose Down
DROP TABLE current_weather;