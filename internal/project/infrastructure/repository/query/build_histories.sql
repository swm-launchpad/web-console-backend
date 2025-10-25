-- name: CreateBuildHistory :execresult
INSERT INTO `BUILD_HISTORY` (
    `container_id`,
    `status`,
    `summary`,
    `tekton_event_id`,
    `tekton_pipeline_run_name`,
    `git_commit_hash`,
    `created_at`,
    `started_at`,
    `finished_at`
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateBuildHistory :execresult
UPDATE `BUILD_HISTORY`
SET
    `status` = ?,
    `summary` = ?,
    `tekton_event_id` = ?,
    `tekton_pipeline_run_name` = ?,
    `git_commit_hash` = ?,
    `started_at` = ?,
    `finished_at` = ?
WHERE
    `build_history_id` = ?;

-- name: FindBuildHistoryByID :one
SELECT
    `build_history_id`,
    `container_id`,
    `status`,
    `summary`,
    `tekton_event_id`,
    `tekton_pipeline_run_name`,
    `git_commit_hash`,
    `created_at`,
    `started_at`,
    `finished_at`
FROM `BUILD_HISTORY`
WHERE `build_history_id` = ?
LIMIT 1;

-- name: FindLatestBuildHistoryByContainerID :one
SELECT
    `build_history_id`,
    `container_id`,
    `status`,
    `summary`,
    `tekton_event_id`,
    `tekton_pipeline_run_name`,
    `git_commit_hash`,
    `created_at`,
    `started_at`,
    `finished_at`
FROM `BUILD_HISTORY`
WHERE `container_id` = ?
ORDER BY `created_at` DESC, `build_history_id` DESC
LIMIT 1;

-- name: FindBuildHistoriesByContainerID :many
SELECT
    `build_history_id`,
    `container_id`,
    `status`,
    `summary`,
    `tekton_event_id`,
    `tekton_pipeline_run_name`,
    `git_commit_hash`,
    `created_at`,
    `started_at`,
    `finished_at`
FROM `BUILD_HISTORY`
WHERE `container_id` = ?
ORDER BY `created_at` DESC, `build_history_id` DESC
LIMIT ? OFFSET ?;

-- name: FindBuildHistoryByTektonPipelineRunName :one
SELECT
    `build_history_id`,
    `container_id`,
    `status`,
    `summary`,
    `tekton_event_id`,
    `tekton_pipeline_run_name`,
    `git_commit_hash`,
    `created_at`,
    `started_at`,
    `finished_at`
FROM `BUILD_HISTORY`
WHERE `tekton_pipeline_run_name` = ?
LIMIT 1;

-- name: FindActiveBuildHistoriesByContainerID :many
-- Returns all non-completed build histories for a container.
-- Includes: untracked, running, backend_tracking_lost (recoverable states)
-- Excludes: success, failed, cancelled, skipped, backend_trigger_failed, backend_tracking_failed (terminal states)
SELECT
    `build_history_id`,
    `container_id`,
    `status`,
    `summary`,
    `tekton_event_id`,
    `tekton_pipeline_run_name`,
    `git_commit_hash`,
    `created_at`,
    `started_at`,
    `finished_at`
FROM `BUILD_HISTORY`
WHERE `container_id` = ?
AND `status` IN ('untracked', 'running', 'backend_tracking_lost')
ORDER BY `created_at` DESC, `build_history_id` DESC;
