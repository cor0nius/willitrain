-- name: CreateHourlyForecast :one
INSERT INTO hourly_forecasts (
    id,
    location_id,
    source_api,
    forecast_datetime_utc,
    temperature_c,
    humidity,
    wind_speed_kmh,
    precipitation_mm
)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetHourlyForecastAtLocationAndTime :many
SELECT * FROM hourly_forecasts WHERE location_id=$1, forecast_datetime_utc=$2;

-- name: GetHourlyForecastAtLocationAndTimeFromAPI :one
SELECT * FROM hourly_forecasts WHERE location_id=$1, forecast_datetime_utc=$2, source_api=$3;

-- name: UpdateHourlyForecast