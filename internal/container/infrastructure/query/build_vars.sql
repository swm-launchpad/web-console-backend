-- Build Variables CRUD

-- name: CreateBuildVar :execresult
INSERT INTO BUILD_VARS (
    container_id, `key`, value,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?);

-- name: GetBuildVarsByContainerID :many
SELECT
    container_id, `key`, value,
    created_at, updated_at, build_var_id
FROM BUILD_VARS
WHERE container_id = ?
ORDER BY `key` ASC;

-- name: GetBuildVarByKey :one
SELECT
    container_id, `key`, value,
    created_at, updated_at, build_var_id
FROM BUILD_VARS
WHERE container_id = ? AND `key` = ?;

-- name: UpdateBuildVar :execresult
UPDATE BUILD_VARS SET
    value = ?,
    updated_at = ?
WHERE container_id = ? AND `key` = ?;

-- name: DeleteBuildVar :execresult
DELETE FROM BUILD_VARS
WHERE container_id = ? AND `key` = ?;

-- name: DeleteBuildVarsByContainerID :execresult
DELETE FROM BUILD_VARS
WHERE container_id = ?;

-- name: CountBuildVarsByContainerID :one
SELECT COUNT(*) as total FROM BUILD_VARS WHERE container_id = ?;
