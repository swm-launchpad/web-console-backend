-- Projects CRUD

-- name: CreateProject :execresult
INSERT INTO PROJECTS (
    name, slug, fqdn, status, plan,
    cpu_limit, memory_limit, disk_limit, traffic_limit,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetProjectByID :one
SELECT
    project_id, name, slug, fqdn, status, plan,
    cpu_limit, memory_limit, disk_limit, traffic_limit,
    created_at, updated_at, deleted_at, is_deleted
FROM PROJECTS
WHERE project_id = ? AND is_deleted = FALSE;

-- name: GetProjectByIDForUpdate :one
SELECT
    project_id, name, slug, fqdn, status, plan,
    cpu_limit, memory_limit, disk_limit, traffic_limit,
    created_at, updated_at, deleted_at, is_deleted
FROM PROJECTS
WHERE project_id = ? AND is_deleted = FALSE
FOR UPDATE;

-- name: GetProjectBySlug :one
SELECT
    project_id, name, slug, fqdn, status, plan,
    cpu_limit, memory_limit, disk_limit, traffic_limit,
    created_at, updated_at, deleted_at, is_deleted
FROM PROJECTS
WHERE slug = ? AND is_deleted = FALSE;

-- name: UpdateProject :execresult
UPDATE PROJECTS SET
    name = ?, fqdn = ?, status = ?, plan = ?,
    cpu_limit = ?, memory_limit = ?, disk_limit = ?, traffic_limit = ?,
    updated_at = ?
WHERE project_id = ? AND is_deleted = FALSE;

-- name: DeleteProject :execresult
UPDATE PROJECTS SET
    is_deleted = TRUE,
    deleted_at = ?,
    updated_at = ?
WHERE project_id = ?;

-- name: ListProjects :many
SELECT
    project_id, name, slug, fqdn, status, plan,
    cpu_limit, memory_limit, disk_limit, traffic_limit,
    created_at, updated_at, deleted_at, is_deleted
FROM PROJECTS
WHERE is_deleted = FALSE
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListProjectsByUserID :many
SELECT DISTINCT
    p.project_id, p.name, p.slug, p.fqdn, p.status, p.plan,
    p.cpu_limit, p.memory_limit, p.disk_limit, p.traffic_limit,
    p.created_at, p.updated_at, p.deleted_at, p.is_deleted
FROM PROJECTS p
JOIN PROJECT_USER pu ON p.project_id = pu.project_id
WHERE pu.user_id = ?
AND p.is_deleted = FALSE
AND pu.is_deleted = FALSE
ORDER BY p.created_at DESC;

-- name: CountProjects :one
SELECT COUNT(*) as total FROM PROJECTS WHERE is_deleted = FALSE;

-- name: ExistsBySlug :one
SELECT EXISTS(SELECT 1 FROM PROJECTS WHERE slug = ? AND is_deleted = FALSE) as project_exists;

-- name: ExistsByNameAndUserID :one
SELECT EXISTS(
    SELECT 1
    FROM PROJECTS p
    WHERE p.name = ?
    AND p.is_deleted = FALSE
    AND EXISTS (
        SELECT 1
        FROM PROJECT_USER pu
        WHERE pu.project_id = p.project_id
        AND pu.user_id = ?
        AND pu.is_deleted = FALSE
    )
) as project_exists;

-- ProjectUsers CRUD

-- name: CreateProjectUser :execresult
INSERT INTO PROJECT_USER (
    project_id, user_id, role,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?);

-- name: GetProjectUsersByProjectID :many
SELECT
    project_user_id, project_id, user_id, role,
    created_at, updated_at, deleted_at, is_deleted
FROM PROJECT_USER
WHERE project_id = ? AND is_deleted = FALSE
ORDER BY created_at ASC;

-- name: GetAllProjectUsersByProjectID :many
SELECT
    project_user_id, project_id, user_id, role,
    created_at, updated_at, deleted_at, is_deleted
FROM PROJECT_USER
WHERE project_id = ?
ORDER BY created_at ASC;

-- name: GetProjectUsersByProjectIDs :many
SELECT
    project_user_id, project_id, user_id, role,
    created_at, updated_at, deleted_at, is_deleted
FROM PROJECT_USER
WHERE project_id IN (sqlc.slice('project_ids')) AND is_deleted = FALSE
ORDER BY project_id, created_at ASC;

-- name: UpdateProjectUser :execresult
UPDATE PROJECT_USER SET
    role = ?, updated_at = ?
WHERE project_id = ? AND user_id = ? AND is_deleted = FALSE;

-- name: DeleteProjectUser :execresult
UPDATE PROJECT_USER SET
    is_deleted = TRUE,
    deleted_at = ?,
    updated_at = ?
WHERE project_id = ? AND user_id = ?;

-- name: RestoreProjectUser :execresult
UPDATE PROJECT_USER SET
    is_deleted = FALSE,
    deleted_at = NULL,
    role = ?,
    updated_at = ?
WHERE project_id = ? AND user_id = ?;

-- name: DeleteProjectUsersByProjectID :execresult
UPDATE PROJECT_USER SET
    is_deleted = TRUE,
    deleted_at = ?,
    updated_at = ?
WHERE project_id = ?;

-- name: GetProjectUserRole :one
SELECT role
FROM PROJECT_USER
WHERE project_id = ? AND user_id = ? AND is_deleted = FALSE;

-- name: HardDeleteProjectUsersByProjectID :execresult
DELETE FROM PROJECT_USER
WHERE project_id = ?;