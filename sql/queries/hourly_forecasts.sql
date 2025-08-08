-- name: CreateHourlyForecast :one
INSERT INTO hourly_forecasts (
    id,
    location_id,
    source_api,
    updated_at,
    forecast_datetime_utc,
    temperature_c,
    humidity,
    wind_speed_kmh,
    precipitation_mm,
    precipitation_chance_percent,
    condition_text 
)
VALUES (gen_random_uuid(), $1, $2, NOW(), $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetHourlyForecastAtLocationAndTime :many
SELECT * FROM hourly_forecasts WHERE location_id=$1 AND forecast_datetime_utc=$2;

-- name: GetAllHourlyForecastsAtLocation :many
SELECT * FROM hourly_forecasts WHERE location_id=$1;

-- name: GetHourlyForecastAtLocationAndTimeFromAPI :one
SELECT * FROM hourly_forecasts WHERE location_id=$1 AND forecast_datetime_utc=$2 AND source_api=$3;

-- name: UpdateHourlyForecast :one
UPDATE hourly_forecasts
SET updated_at=NOW(), forecast_datetime_utc=$2, temperature_c=$3, humidity=$4, wind_speed_kmh=$5, precipitation_mm=$6, precipitation_chance_percent=$7, condition_text=$8
WHERE id=$1
RETURNING *;

-- name: DeleteHourlyForecastsAtLocation :exec
DELETE FROM hourly_forecasts WHERE location_id=$1;

-- name: DeleteHourlyForecastsAtLocationFromAPI :exec
DELETE FROM hourly_forecasts WHERE location_id=$1 AND source_api=$2;

-- name: DeleteHourlyForecastsFromAPI :exec
DELETE FROM hourly_forecasts WHERE source_api=$1;

-- name: DeleteAllHourlyForecasts :exec
DELETE FROM hourly_forecasts;