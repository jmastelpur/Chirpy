-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (gen_random_uuid(), NOW(), NOW(), $1, $2)
RETURNING *;


-- name: GetChirpsByUserID :many
/* retrieve all chirps by a specific user ID Order them by created_at in ascending order. */
SELECT * FROM chirps
WHERE user_id = $1
ORDER BY created_at ASC;

-- name: GetChirps :many
/* all chirps in the database. Order them by created_at in ascending order. */
SELECT * FROM chirps
ORDER BY created_at ASC;

-- name: GetChirpsByUserIDDesc :many
/* retrieve all chirps by a specific user ID Order them by created_at in descending order. */
SELECT * FROM chirps
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetChirpsDesc :many
/* all chirps in the database. Order them by created_at in descending order. */
SELECT * FROM chirps
ORDER BY created_at DESC;


-- name: GetChirpByID :one
/* retrieve a single chirp by its ID */
SELECT * FROM chirps
WHERE id = $1;


-- name: DeleteChirp :exec
/* delete a single chirp by its ID */
DELETE FROM chirps
WHERE id = $1;