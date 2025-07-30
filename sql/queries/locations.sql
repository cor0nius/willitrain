-- name: CreateLocation :one
INSERT INTO locations (id, city_name, latitude, longitude, country_code)
VALUES (gen_random_uuid(), $1, $2, $3, $4)
RETURNING *;

-- name: ListLocations :many
SELECT * FROM locations ORDER BY city_name ASC;

-- name: GetLocationByName :one
SELECT * FROM locations WHERE city_name=$1;

-- name: GetLocationByCoordinates :one
SELECT * FROM locations WHERE latitude=$1 AND longitude=$2;

-- name: DeleteLocation :exec
DELETE FROM locations WHERE id=$1;

-- name: DeleteAllLocations :exec
DELETE FROM locations;