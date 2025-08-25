-- CreateLocation inserts a new location record.
-- name: CreateLocation :one
INSERT INTO locations (id, city_name, latitude, longitude, country_code)
VALUES (gen_random_uuid(), $1, $2, $3, $4)
RETURNING *;

-- ListLocations retrieves all locations, ordered by city name.
-- name: ListLocations :many
SELECT * FROM locations ORDER BY city_name ASC;

-- GetLocationByName retrieves a location by its city name.
-- name: GetLocationByName :one
SELECT * FROM locations WHERE city_name=$1;

-- GetLocationByCoordinates retrieves a location by its latitude and longitude.
-- name: GetLocationByCoordinates :one
SELECT * FROM locations WHERE latitude=$1 AND longitude=$2;

-- DeleteLocation deletes a location by its ID.
-- name: DeleteLocation :exec
DELETE FROM locations WHERE id=$1;

-- DeleteAllLocations deletes all locations from the database.
-- name: DeleteAllLocations :exec
DELETE FROM locations;

-- UpdateTimezone updates the timezone for a specific location.
-- name: UpdateTimezone :exec
UPDATE locations
SET timezone = $2
WHERE id = $1;