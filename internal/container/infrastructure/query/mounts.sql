-- Volume Mounts CRUD

-- name: CreateMount :execresult
INSERT INTO MOUNTS (
    container_id, volume_id, mount_path,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?);

-- name: GetMountsByContainerID :many
SELECT
    container_id, volume_id, mount_path,
    created_at, updated_at
FROM MOUNTS
WHERE container_id = ?
ORDER BY mount_path ASC;

-- name: GetMountByVolume :one
SELECT
    container_id, volume_id, mount_path,
    created_at, updated_at
FROM MOUNTS
WHERE container_id = ? AND volume_id = ?;

-- name: DeleteMount :execresult
DELETE FROM MOUNTS
WHERE container_id = ? AND volume_id = ?;

-- name: DeleteMountsByContainerID :execresult
DELETE FROM MOUNTS
WHERE container_id = ?;

-- name: CountMountsByContainerID :one
SELECT COUNT(*) as total FROM MOUNTS WHERE container_id = ?;

-- name: ExistsMountByVolume :one
SELECT EXISTS(
    SELECT 1 FROM MOUNTS
    WHERE container_id = ? AND volume_id = ?
) as `exists`;
