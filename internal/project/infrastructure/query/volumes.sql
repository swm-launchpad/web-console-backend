-- Volumes CRUD (Independent Aggregate Root)

-- name: CreateVolume :execresult
INSERT INTO VOLUMES (
    project_id, name, capacity,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?);

-- name: GetVolumeByID :one
SELECT
    volume_id, project_id, name, capacity,
    created_at, updated_at
FROM VOLUMES
WHERE volume_id = ?;

-- name: GetVolumesByProjectID :many
SELECT
    volume_id, project_id, name, capacity,
    created_at, updated_at
FROM VOLUMES
WHERE project_id = ?
ORDER BY created_at ASC;

-- name: GetVolumeByName :one
SELECT
    volume_id, project_id, name, capacity,
    created_at, updated_at
FROM VOLUMES
WHERE project_id = ? AND name = ?;

-- name: ExistsVolumeByName :one
SELECT EXISTS(
    SELECT 1 FROM VOLUMES
    WHERE project_id = ? AND name = ?
) as volume_exists;

-- name: UpdateVolume :execresult
UPDATE VOLUMES SET
    name = ?, capacity = ?,
    updated_at = ?
WHERE volume_id = ?;

-- name: DeleteVolume :execresult
DELETE FROM VOLUMES WHERE volume_id = ?;

-- name: ListVolumes :many
SELECT
    volume_id, project_id, name, capacity,
    created_at, updated_at
FROM VOLUMES
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountVolumes :one
SELECT COUNT(*) as total FROM VOLUMES;

-- name: CountVolumesByProjectID :one
SELECT COUNT(*) as total FROM VOLUMES WHERE project_id = ?;