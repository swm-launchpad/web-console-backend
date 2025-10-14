-- name: CreateDeployment :execresult
INSERT INTO `DEPLOYMENTS` (
    `project_id`,
    `status`,
    `summary`,
    `tekton_event_id`,
    `tekton_pipeline_run_name`,
    `created_at`,
    `started_at`,
    `finished_at`
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateDeployment :execresult
UPDATE `DEPLOYMENTS`
SET
    `status` = ?,
    `summary` = ?,
    `tekton_event_id` = ?,
    `tekton_pipeline_run_name` = ?,
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
    `tekton_event_id`,
    `tekton_pipeline_run_name`,
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
    `tekton_event_id`,
    `tekton_pipeline_run_name`,
    `created_at`,
    `started_at`,
    `finished_at`
FROM `DEPLOYMENTS`
WHERE `project_id` = ?
ORDER BY `created_at` DESC, `deployment_id` DESC
LIMIT 1;

-- name: FindDeploymentsByProjectID :many
SELECT
    `deployment_id`,
    `project_id`,
    `status`,
    `summary`,
    `tekton_event_id`,
    `tekton_pipeline_run_name`,
    `created_at`,
    `started_at`,
    `finished_at`
FROM `DEPLOYMENTS`
WHERE `project_id` = ?
ORDER BY `created_at` DESC, `deployment_id` DESC
LIMIT ? OFFSET ?;

-- name: FindDeploymentByTektonPipelineRunName :one
SELECT
    `deployment_id`,
    `project_id`,
    `status`,
    `summary`,
    `tekton_event_id`,
    `tekton_pipeline_run_name`,
    `created_at`,
    `started_at`,
    `finished_at`
FROM `DEPLOYMENTS`
WHERE `tekton_pipeline_run_name` = ?
LIMIT 1;

-- name: FindActiveDeploymentsByProjectID :many
SELECT
    `deployment_id`,
    `project_id`,
    `status`,
    `summary`,
    `tekton_event_id`,
    `tekton_pipeline_run_name`,
    `created_at`,
    `started_at`,
    `finished_at`
FROM `DEPLOYMENTS`
WHERE `project_id` = ?
AND `status` IN ('untracked', 'running')
ORDER BY `created_at` DESC, `deployment_id` DESC;
