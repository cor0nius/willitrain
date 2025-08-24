-- name: CreateHourlyForecast :one
INSERT INTO hourly_forecasts (
    id,
    location_id,
    source_api,
    forecast_datetime_utc,
    updated_at,
    temperature_c,
    humidity,
    wind_speed_kmh,
    precipitation_mm,
    precipitation_chance_percent,
    condition_text 
)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetHourlyForecastAtLocationAndTime :many
SELECT * FROM hourly_forecasts WHERE location_id=$1 AND forecast_datetime_utc=$2;

-- name: GetAllHourlyForecastsAtLocation :many
SELECT * FROM hourly_forecasts WHERE location_id=$1;

-- name: GetHourlyForecastAtLocationAndTimeFromAPI :one
SELECT * FROM hourly_forecasts WHERE location_id=$1 AND forecast_datetime_utc=$2 AND source_api=$3;

-- name: UpdateHourlyForecast :one
UPDATE hourly_forecasts
SET updated_at=$2, forecast_datetime_utc=$3, temperature_c=$4, humidity=$5, wind_speed_kmh=$6, precipitation_mm=$7, precipitation_chance_percent=$8, condition_text=$9
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

-- name: GetUpcomingHourlyForecastsAtLocation :many
SELECT * FROM hourly_forecasts
WHERE location_id = $1 AND forecast_datetime_utc >= $2
ORDER BY forecast_datetime_utc ASC;
