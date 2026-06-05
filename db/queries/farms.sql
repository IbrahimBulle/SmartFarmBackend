-- name: CreateFarm :one
INSERT INTO farms (
    user_id,
    name,
    location,
    crop_type,
    size_acres
) VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListFarmsByUser :many
SELECT *
FROM farms
WHERE user_id = ?
ORDER BY created_at DESC;

-- name: GetFarm :one
SELECT *
FROM farms
WHERE id = ?;

-- name: DeleteFarm :exec
DELETE FROM farms
WHERE id = ?;