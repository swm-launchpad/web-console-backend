-- Environment Variables CRUD

-- name: CreateEnvVar :execresult
INSERT INTO ENV_VARS (
    container_id, `key`, value,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?);

-- name: GetEnvVarsByContainerID :many
SELECT
    env_var_id, container_id, `key`, value,
    created_at, updated_at
FROM ENV_VARS
WHERE container_id = ?
ORDER BY `key` ASC;

-- name: GetEnvVarByKey :one
SELECT
    env_var_id, container_id, `key`, value,
    created_at, updated_at
FROM ENV_VARS
WHERE container_id = ? AND `key` = ?;

-- name: UpdateEnvVar :execresult
UPDATE ENV_VARS SET
    value = ?,
    updated_at = ?
WHERE container_id = ? AND `key` = ?;

-- name: DeleteEnvVar :execresult
DELETE FROM ENV_VARS
WHERE container_id = ? AND `key` = ?;

-- name: DeleteEnvVarsByContainerID :execresult
DELETE FROM ENV_VARS
WHERE container_id = ?;

-- name: CountEnvVarsByContainerID :one
SELECT COUNT(*) as total FROM ENV_VARS WHERE container_id = ?;
