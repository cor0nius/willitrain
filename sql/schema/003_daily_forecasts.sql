-- +goose Up
CREATE TABLE daily_forecasts (
    id UUID PRIMARY KEY,
    location_id UUID REFERENCES locations(id) ON DELETE CASCADE NOT NULL,
    source_api TEXT NOT NULL,
    forecast_date DATE NOT NULL,
    min_temp_c FLOAT,
    max_temp_c FLOAT,
    avg_temp_c FLOAT,
    precipitation_mm FLOAT,
    chance_of_rain_percent INT
);

-- +goose Down
DROP TABLE daily_forecasts;