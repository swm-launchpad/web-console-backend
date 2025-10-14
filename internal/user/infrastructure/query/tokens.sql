-- name: CreateVerificationToken :exec
INSERT INTO verification_tokens (
    user_id,
    token,
    token_type,
    expires_at,
    created_at
) VALUES (
    ?, ?, ?, ?, ?
);

-- name: FindTokenByToken :one
SELECT
    token_id,
    user_id,
    token,
    token_type,
    expires_at,
    used_at,
    created_at
FROM verification_tokens
WHERE token = ?
LIMIT 1;

-- name: MarkTokenAsUsed :exec
UPDATE verification_tokens
SET used_at = ?
WHERE token_id = ?;

-- name: DeleteUserTokensByType :exec
DELETE FROM verification_tokens
WHERE user_id = ? AND token_type = ?;

-- name: DeleteExpiredTokens :exec
DELETE FROM verification_tokens
WHERE expires_at < NOW();

-- name: FindLatestTokenByUserAndType :one
SELECT
    token_id,
    user_id,
    token,
    token_type,
    expires_at,
    used_at,
    created_at
FROM verification_tokens
WHERE user_id = ? AND token_type = ?
ORDER BY created_at DESC
LIMIT 1;
