-- CreateLocationAlias creates a new alias for a location.
-- name: CreateLocationAlias :one
INSERT INTO location_aliases (alias, location_id)
VALUES ($1, $2)
RETURNING *;

-- GetLocationByAlias retrieves a location's details by its alias.
-- name: GetLocationByAlias :one
SELECT l.* FROM locations l JOIN location_aliases la ON l.id = la.location_id
WHERE la.alias = $1;