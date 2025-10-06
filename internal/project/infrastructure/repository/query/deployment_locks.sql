-- Deployment Locks CRUD

-- name: GetDeploymentLock :one
SELECT
    project_id, token, expires_at
FROM DEPLOYMENT_LOCKS
WHERE project_id = ?;

-- name: InsertNewLock :execresult
INSERT INTO DEPLOYMENT_LOCKS (
    project_id, token, expires_at
) VALUES (?, ?, ?);

-- name: UpdateExpiredLock :execresult
UPDATE DEPLOYMENT_LOCKS SET
    token = token + 1,
    expires_at = ?
WHERE project_id = ? AND expires_at <= NOW();

-- name: RenewDeploymentLock :execresult
UPDATE DEPLOYMENT_LOCKS SET
    expires_at = ?
WHERE project_id = ? AND token <= ? AND expires_at > NOW();

-- name: ReleaseDeploymentLock :execresult
UPDATE DEPLOYMENT_LOCKS SET
    expires_at = NOW()
WHERE project_id = ? AND token <= ?;
