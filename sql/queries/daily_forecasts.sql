-- CreateDailyForecast inserts a new daily forecast record.
-- name: CreateDailyForecast :one
INSERT INTO daily_forecasts (
    id,
    location_id,
    source_api,
    forecast_date,
    updated_at,
    min_temp_c,
    max_temp_c,
    precipitation_mm,
    precipitation_chance_percent,
    wind_speed_kmh,
    humidity 
)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- GetDailyForecastAtLocationAndDate retrieves all daily forecasts for a specific location and date.
-- name: GetDailyForecastAtLocationAndDate :many
SELECT * FROM daily_forecasts WHERE location_id=$1 AND forecast_date=$2;

-- GetAllDailyForecastsAtLocation retrieves all daily forecasts for a specific location.
-- name: GetAllDailyForecastsAtLocation :many
SELECT * FROM daily_forecasts WHERE location_id=$1;

-- GetDailyForecastAtLocationAndDateFromAPI retrieves the daily forecast for a specific location, date, and API source.
-- name: GetDailyForecastAtLocationAndDateFromAPI :one
SELECT * FROM daily_forecasts WHERE location_id=$1 AND forecast_date=$2 AND source_api=$3;

-- UpdateDailyForecast updates an existing daily forecast record.
-- name: UpdateDailyForecast :one
UPDATE daily_forecasts
SET updated_at=$2, forecast_date=$3, min_temp_c=$4, max_temp_c=$5, precipitation_mm=$6, precipitation_chance_percent=$7, wind_speed_kmh=$8, humidity=$9
WHERE id=$1
RETURNING *;

-- DeleteDailyForecastsAtLocation deletes all daily forecasts for a specific location.
-- name: DeleteDailyForecastsAtLocation :exec
DELETE FROM daily_forecasts WHERE location_id=$1;

-- DeleteDailyForecastsAtLocationFromAPI deletes all daily forecasts for a specific location and API source.
-- name: DeleteDailyForecastsAtLocationFromAPI :exec
DELETE FROM daily_forecasts WHERE location_id=$1 AND source_api=$2;

-- DeleteDailyForecastsFromApi deletes all daily forecasts from a specific API source.
-- name: DeleteDailyForecastsFromApi :exec
DELETE FROM daily_forecasts WHERE source_api=$1;

-- DeleteAllDailyForecasts deletes all daily forecasts from the database.
-- name: DeleteAllDailyForecasts :exec
DELETE FROM daily_forecasts;

-- GetUpcomingDailyForecastsAtLocation retrieves all upcoming daily forecasts for a specific location.
-- name: GetUpcomingDailyForecastsAtLocation :many
SELECT * FROM daily_forecasts
WHERE location_id = $1 AND forecast_date >= $2
ORDER BY forecast_date ASC;
