-- +goose Up
CREATE TABLE daily_forecasts (
    id UUID PRIMARY KEY,
    location_id UUID REFERENCES locations(id) ON DELETE CASCADE NOT NULL,
    source_api TEXT NOT NULL,
    forecast_date DATE NOT NULL,
    min_temp_c FLOAT,
    max_temp_c FLOAT,
    precipitation_mm FLOAT,
    precipitation_chance_percent INT,
    wind_speed_kmh FLOAT,
    humidity INT
);

-- +goose Down
DROP TABLE daily_forecasts;