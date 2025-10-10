-- Deployment Locks CRUD

-- name: GetDeploymentLock :one
SELECT
    project_id, token, expires_at
FROM DEPLOYMENT_LOCKS
WHERE project_id = ?;

-- name: AcquireOrUpdateLock :execresult
-- Atomically acquire a new lock or update an expired lock
-- Returns RowsAffected: 1 (INSERT new), 2 (UPDATE expired), 0 (active lock exists)
INSERT INTO DEPLOYMENT_LOCKS (
    project_id, token, expires_at
) VALUES (?, 1, ?)
ON DUPLICATE KEY UPDATE
    token = IF(expires_at <= NOW(), token + 1, token),
    expires_at = IF(expires_at <= NOW(), VALUES(expires_at), expires_at);

-- name: RenewDeploymentLock :execresult
UPDATE DEPLOYMENT_LOCKS SET
    expires_at = ?
WHERE project_id = ? AND token = ? AND expires_at > NOW();

-- name: ReleaseDeploymentLock :execresult
UPDATE DEPLOYMENT_LOCKS SET
    expires_at = NOW()
WHERE project_id = ? AND token = ?;
