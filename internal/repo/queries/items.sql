-- name: ListItems :many
SELECT *
FROM items
ORDER BY created_at DESC;

-- name: GetItem :one
SELECT *
FROM items
WHERE id = $1;

-- name: CreateItem :one
INSERT INTO items (title)
VALUES ($1)
RETURNING *;

-- name: DeleteItem :execrows
DELETE
FROM items
WHERE id = $1;
