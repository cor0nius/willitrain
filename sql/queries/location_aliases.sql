-- name: CreateLocationAlias :one
INSERT INTO location_aliases (alias, location_id)
VALUES ($1, $2)
RETURNING *;

-- name: GetLocationByAlias :one
SELECT l.* FROM locations l JOIN location_aliases la ON l.id = la.location_id
WHERE la.alias = $1;