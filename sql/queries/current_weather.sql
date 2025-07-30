-- name: CreateCurrentWeather :one
INSERT INTO current_weather (
    id,
    location_id,
    source_api,
    updated_at,
    temperature_c,
    humidity,
    wind_speed_kmh,
    precipitation_mm,
    condition_text
)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetCurrentWeatherAtLocation :many
SELECT * FROM current_weather WHERE location_id=$1;

-- name: GetCurrentWeatherAtLocationFromAPI :one
SELECT * FROM current_weather WHERE location_id=$1 AND source_api=$2;

-- name: UpdateCurrentWeather :one
UPDATE current_weather
SET updated_at=$2, temperature_c=$3, humidity=$4, wind_speed_kmh=$5, precipitation_mm=$6, condition_text=$7
WHERE id=$1
RETURNING *;

-- name: DeleteCurrentWeatherAtLocation :exec
DELETE FROM current_weather WHERE location_id=$1;

-- name: DeleteCurrentWeatherAtLocationFromAPI :exec
DELETE FROM current_weather WHERE location_id=$1 AND source_api=$2;

-- name: DeleteAllCurrentWeatherFromAPI :exec
DELETE FROM current_weather WHERE source_api=$1;

-- name: DeleteAllCurrentWeather :exec
DELETE FROM current_weather;