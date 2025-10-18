-- name: CreateGitHubInstallation :execresult
INSERT INTO GITHUB_INSTALLATIONS (
    installation_id, user_id, account_login, account_type, status,
    cached_token, token_expires_at,
    is_deleted, deleted_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetGitHubInstallationByID :one
SELECT
    installation_id, user_id, account_login, account_type, status,
    cached_token, token_expires_at,
    is_deleted, deleted_at, created_at, updated_at
FROM GITHUB_INSTALLATIONS
WHERE installation_id = ? AND is_deleted = FALSE AND status = 'active';

-- name: GetGitHubInstallationsByUserID :many
SELECT
    installation_id, user_id, account_login, account_type, status,
    cached_token, token_expires_at,
    is_deleted, deleted_at, created_at, updated_at
FROM GITHUB_INSTALLATIONS
WHERE user_id = ? AND is_deleted = FALSE AND status = 'active'
ORDER BY created_at DESC;

-- name: UpdateGitHubInstallation :execresult
UPDATE GITHUB_INSTALLATIONS SET
    cached_token = ?,
    token_expires_at = ?,
    updated_at = ?
WHERE installation_id = ?;

-- name: DeleteGitHubInstallation :execresult
UPDATE GITHUB_INSTALLATIONS SET
    is_deleted = TRUE,
    deleted_at = ?,
    updated_at = ?,
    cached_token = NULL,
    token_expires_at = NULL
WHERE installation_id = ?;

-- name: MarkInstallationAsRevoked :execresult
UPDATE GITHUB_INSTALLATIONS SET
    status = 'revoked',
    cached_token = NULL,
    token_expires_at = NULL,
    updated_at = ?
WHERE installation_id = ?;

-- name: ExistsByInstallationID :one
SELECT EXISTS(SELECT 1 FROM GITHUB_INSTALLATIONS WHERE installation_id = ? AND is_deleted = FALSE AND status = 'active') as installation_exists;

-- name: FindInstallationByIDIncludingRevoked :one
SELECT
    installation_id, user_id, account_login, account_type, status,
    cached_token, token_expires_at,
    is_deleted, deleted_at, created_at, updated_at
FROM GITHUB_INSTALLATIONS
WHERE installation_id = ?;

-- name: ReactivateInstallation :execresult
UPDATE GITHUB_INSTALLATIONS SET
    status = 'active',
    account_login = ?,
    account_type = ?,
    is_deleted = FALSE,
    deleted_at = NULL,
    updated_at = ?
WHERE installation_id = ?;
