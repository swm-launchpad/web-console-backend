-- name: CreateOAuthState :execresult
INSERT INTO OAUTH_STATES (
    state, user_id, installation_id, expires_at, created_at, consumed_at
) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetOAuthStateByState :one
SELECT
    state, user_id, installation_id, expires_at, created_at, consumed_at
FROM OAUTH_STATES
WHERE state = ? LIMIT 1;

-- name: MarkOAuthStateAsConsumed :execresult
UPDATE OAUTH_STATES SET
    consumed_at = ?,
    installation_id = ?
WHERE state = ? AND consumed_at IS NULL;

-- name: DeleteExpiredOAuthStates :execresult
DELETE FROM OAUTH_STATES
WHERE expires_at < NOW() OR consumed_at IS NOT NULL;
