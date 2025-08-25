-- CreateHourlyForecast inserts a new hourly forecast record.
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

-- GetHourlyForecastAtLocationAndTime retrieves all hourly forecasts for a specific location and time.
-- name: GetHourlyForecastAtLocationAndTime :many
SELECT * FROM hourly_forecasts WHERE location_id=$1 AND forecast_datetime_utc=$2;

-- GetAllHourlyForecastsAtLocation retrieves all hourly forecasts for a specific location.
-- name: GetAllHourlyForecastsAtLocation :many
SELECT * FROM hourly_forecasts WHERE location_id=$1;

-- GetHourlyForecastAtLocationAndTimeFromAPI retrieves the hourly forecast for a specific location, time, and API source.
-- name: GetHourlyForecastAtLocationAndTimeFromAPI :one
SELECT * FROM hourly_forecasts WHERE location_id=$1 AND forecast_datetime_utc=$2 AND source_api=$3;

-- UpdateHourlyForecast updates an existing hourly forecast record.
-- name: UpdateHourlyForecast :one
UPDATE hourly_forecasts
SET updated_at=$2, forecast_datetime_utc=$3, temperature_c=$4, humidity=$5, wind_speed_kmh=$6, precipitation_mm=$7, precipitation_chance_percent=$8, condition_text=$9
WHERE id=$1
RETURNING *;

-- DeleteHourlyForecastsAtLocation deletes all hourly forecasts for a specific location.
-- name: DeleteHourlyForecastsAtLocation :exec
DELETE FROM hourly_forecasts WHERE location_id=$1;

-- DeleteHourlyForecastsAtLocationFromAPI deletes all hourly forecasts for a specific location and API source.
-- name: DeleteHourlyForecastsAtLocationFromAPI :exec
DELETE FROM hourly_forecasts WHERE location_id=$1 AND source_api=$2;

-- DeleteHourlyForecastsFromAPI deletes all hourly forecasts from a specific API source.
-- name: DeleteHourlyForecastsFromAPI :exec
DELETE FROM hourly_forecasts WHERE source_api=$1;

-- DeleteAllHourlyForecasts deletes all hourly forecasts from the database.
-- name: DeleteAllHourlyForecasts :exec
DELETE FROM hourly_forecasts;

-- GetUpcomingHourlyForecastsAtLocation retrieves all upcoming hourly forecasts for a specific location.
-- name: GetUpcomingHourlyForecastsAtLocation :many
SELECT * FROM hourly_forecasts
WHERE location_id = $1 AND forecast_datetime_utc >= $2
ORDER BY forecast_datetime_utc ASC;
