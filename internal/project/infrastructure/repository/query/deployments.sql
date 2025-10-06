-- name: CreateDeployment :execresult
INSERT INTO `DEPLOYMENTS` (
    `project_id`,
    `status`,
    `summary`,
    `tekton_ref`,
    `created_at`,
    `started_at`,
    `finished_at`
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateDeployment :exec
UPDATE `DEPLOYMENTS`
SET
    `status` = ?,
    `summary` = ?,
    `tekton_ref` = ?,
    `started_at` = ?,
    `finished_at` = ?
WHERE
    `deployment_id` = ?;

-- name: FindDeploymentByID :one
SELECT
    `deployment_id`,
    `project_id`,
    `status`,
    `summary`,
    `tekton_ref`,
    `created_at`,
    `started_at`,
    `finished_at`
FROM `DEPLOYMENTS`
WHERE `deployment_id` = ?
LIMIT 1;

-- name: FindLatestDeploymentByProjectID :one
SELECT
    `deployment_id`,
    `project_id`,
    `status`,
    `summary`,
    `tekton_ref`,
    `created_at`,
    `started_at`,
    `finished_at`
FROM `DEPLOYMENTS`
WHERE `project_id` = ?
ORDER BY `created_at` DESC
LIMIT 1;

-- name: FindDeploymentsByProjectID :many
SELECT
    `deployment_id`,
    `project_id`,
    `status`,
    `summary`,
    `tekton_ref`,
    `created_at`,
    `started_at`,
    `finished_at`
FROM `DEPLOYMENTS`
WHERE `project_id` = ?
ORDER BY `created_at` DESC
LIMIT ? OFFSET ?;
