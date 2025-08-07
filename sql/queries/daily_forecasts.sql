-- name: CreateDailyForecast :one
INSERT INTO daily_forecasts (
    id,
    location_id,
    source_api,
    forecast_date,
    min_temp_c,
    max_temp_c,
    precipitation_mm,
    precipitation_chance_percent,
    wind_speed_kmh,
    humidity
)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetDailyForecastAtLocationAndDate :many
SELECT * FROM daily_forecasts WHERE location_id=$1 AND forecast_date=$2;

-- name: GetDailyForecastAtLocationAndDateFromAPI :one
SELECT * FROM daily_forecasts WHERE location_id=$1 AND forecast_date=$2 AND source_api=$3;

-- name: UpdateDailyForecast :one
UPDATE daily_forecasts
SET forecast_date=$2, min_temp_c=$3, max_temp_c=$4, precipitation_mm=$5, precipitation_chance_percent=$6, wind_speed_kmh=$7, humidity=$8
WHERE id=$1
RETURNING *;

-- name: DeleteDailyForecastsAtLocation :exec
DELETE FROM daily_forecasts WHERE location_id=$1;

-- name: DeleteDailyForecastsAtLocationFromAPI :exec
DELETE FROM daily_forecasts WHERE location_id=$1 AND source_api=$2;

-- name: DeleteDailyForecastsFromApi :exec
DELETE FROM daily_forecasts WHERE source_api=$1;

-- name: DeleteAllDailyForecasts :exec
DELETE FROM daily_forecasts;