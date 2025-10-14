-- Secrets CRUD

-- name: CreateSecret :execresult
INSERT INTO SECRETS (
    container_id, `key`, value,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?);

-- name: GetSecretsByContainerID :many
SELECT
    secret_id, container_id, `key`, value,
    created_at, updated_at
FROM SECRETS
WHERE container_id = ?
ORDER BY `key` ASC;

-- name: GetSecretByKey :one
SELECT
    secret_id, container_id, `key`, value,
    created_at, updated_at
FROM SECRETS
WHERE container_id = ? AND `key` = ?;

-- name: UpdateSecret :execresult
UPDATE SECRETS SET
    value = ?,
    updated_at = ?
WHERE container_id = ? AND `key` = ?;

-- name: DeleteSecret :execresult
DELETE FROM SECRETS
WHERE container_id = ? AND `key` = ?;

-- name: DeleteSecretsByContainerID :execresult
DELETE FROM SECRETS
WHERE container_id = ?;

-- name: CountSecretsByContainerID :one
SELECT COUNT(*) as total FROM SECRETS WHERE container_id = ?;
