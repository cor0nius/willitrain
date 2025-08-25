-- CreateCurrentWeather inserts a new current weather record into the database.
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

-- GetCurrentWeatherAtLocation retrieves all current weather records for a specific location.
-- name: GetCurrentWeatherAtLocation :many
SELECT * FROM current_weather WHERE location_id=$1;

-- GetCurrentWeatherAtLocationFromAPI retrieves the current weather record for a specific location and API source.
-- name: GetCurrentWeatherAtLocationFromAPI :one
SELECT * FROM current_weather WHERE location_id=$1 AND source_api=$2;

-- UpdateCurrentWeather updates an existing current weather record.
-- name: UpdateCurrentWeather :one
UPDATE current_weather
SET updated_at=$2, temperature_c=$3, humidity=$4, wind_speed_kmh=$5, precipitation_mm=$6, condition_text=$7
WHERE id=$1
RETURNING *;

-- DeleteCurrentWeatherAtLocation deletes all current weather records for a specific location.
-- name: DeleteCurrentWeatherAtLocation :exec
DELETE FROM current_weather WHERE location_id=$1;

-- DeleteCurrentWeatherAtLocationFromAPI deletes the current weather record for a specific location and API source.
-- name: DeleteCurrentWeatherAtLocationFromAPI :exec
DELETE FROM current_weather WHERE location_id=$1 AND source_api=$2;

-- DeleteAllCurrentWeatherFromAPI deletes all current weather records from a specific API source.
-- name: DeleteAllCurrentWeatherFromAPI :exec
DELETE FROM current_weather WHERE source_api=$1;

-- DeleteAllCurrentWeather deletes all current weather records from the database.
-- name: DeleteAllCurrentWeather :exec
DELETE FROM current_weather;