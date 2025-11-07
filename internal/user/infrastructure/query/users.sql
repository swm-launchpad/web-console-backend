-- name: CreateUser :execresult
INSERT INTO USERS (
    password_hash, password_updated_at,
    nickname, email, phone, organization,
    status, is_deleted, deleted_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetUserByID :one
SELECT
    user_id, password_hash, password_updated_at,
    nickname, email, phone, organization,
    status, is_deleted, deleted_at, created_at, updated_at
FROM USERS
WHERE user_id = ? AND is_deleted = FALSE;

-- name: GetUserByEmail :one
SELECT
    user_id, password_hash, password_updated_at,
    nickname, email, phone, organization,
    status, is_deleted, deleted_at, created_at, updated_at
FROM USERS
WHERE email = ? AND is_deleted = FALSE;

-- name: UpdateUser :execresult
UPDATE USERS SET
    password_hash = ?, password_updated_at = ?, nickname = ?,
    email = ?, phone = ?, organization = ?,
    status = ?, is_deleted = ?, deleted_at = ?, updated_at = ?
WHERE user_id = ?;

-- name: DeleteUser :execresult
UPDATE USERS SET
    is_deleted = TRUE,
    deleted_at = ?,
    updated_at = ?
WHERE user_id = ?;

-- name: ExistsByEmail :one
SELECT EXISTS(SELECT 1 FROM USERS WHERE email = ? AND is_deleted = FALSE) as user_exists;
