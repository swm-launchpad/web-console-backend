-- Containers CRUD

-- name: CreateContainer :execresult
INSERT INTO CONTAINERS (
    project_id, template_id, name, slug, stable_window,
    template_config, github_installation_id, git_repository_url, git_branch, git_directory_path, git_commit_hash,
    last_built_git_commit_hash, needs_build, cpu_limit, memory_limit,
    monthly_build_time, monthly_build_count, monthly_uptime,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetContainerByID :one
SELECT *
FROM CONTAINERS
WHERE container_id = ? AND is_deleted = FALSE;

-- name: GetContainerByIDForUpdate :one
SELECT *
FROM CONTAINERS
WHERE container_id = ? AND is_deleted = FALSE
FOR UPDATE;

-- name: GetContainerBySlug :one
SELECT *
FROM CONTAINERS
WHERE slug = ? AND is_deleted = FALSE;

-- name: UpdateContainer :execresult
UPDATE CONTAINERS SET
    template_id = ?, name = ?, stable_window = ?,
    template_config = ?, github_installation_id = ?,
    git_repository_url = ?, git_branch = ?, git_directory_path = ?, git_commit_hash = ?,
    last_built_git_commit_hash = ?, needs_build = ?, cpu_limit = ?, memory_limit = ?,
    monthly_build_time = ?, monthly_build_count = ?, monthly_uptime = ?,
    updated_at = ?
WHERE container_id = ? AND is_deleted = FALSE;

-- name: DeleteContainer :execresult
UPDATE CONTAINERS SET
    is_deleted = TRUE,
    deleted_at = ?,
    updated_at = ?
WHERE container_id = ?;

-- name: DeleteContainersByProjectID :execresult
UPDATE CONTAINERS SET
    is_deleted = TRUE,
    deleted_at = ?,
    updated_at = ?
WHERE project_id = ?;

-- name: ListContainersByProjectID :many
SELECT *
FROM CONTAINERS
WHERE project_id = ? AND is_deleted = FALSE
ORDER BY created_at DESC;

-- name: ListContainers :many
SELECT *
FROM CONTAINERS
WHERE is_deleted = FALSE
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountContainers :one
SELECT COUNT(*) as total FROM CONTAINERS WHERE is_deleted = FALSE;

-- name: CountContainersByProjectID :one
SELECT COUNT(*) as total FROM CONTAINERS WHERE project_id = ? AND is_deleted = FALSE;

-- name: CountContainersByTemplateID :one
SELECT COUNT(*) as total FROM CONTAINERS WHERE template_id = ? AND is_deleted = FALSE;

-- name: ExistsBySlug :one
SELECT EXISTS(SELECT 1 FROM CONTAINERS WHERE slug = ? AND is_deleted = FALSE) as container_exists;

-- name: ExistsByNameAndProjectID :one
SELECT EXISTS(SELECT 1 FROM CONTAINERS WHERE project_id = ? AND name = ? AND is_deleted = FALSE) as container_exists;

-- name: GetTotalResourceUsageByProject :one
SELECT
    COALESCE(SUM(cpu_limit), 0) as total_cpu,
    COALESCE(SUM(memory_limit), 0) as total_memory
FROM CONTAINERS
WHERE project_id = ? AND is_deleted = FALSE;
