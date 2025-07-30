-- +goose Up
CREATE TABLE hourly_forecasts (
    id UUID PRIMARY KEY,
    location_id UUID REFERENCES locations(id) ON DELETE CASCADE NOT NULL,
    source_api TEXT NOT NULL,
    forecast_datetime_utc TIMESTAMP NOT NULL,
    temperature_c FLOAT,
    humidity INT,
    wind_speed_kmh FLOAT,
    precipitation_mm FLOAT
);

-- +goose Down
DROP TABLE hourly_forecasts;