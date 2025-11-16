-- name: CreateStatusCheck :execresult
INSERT INTO SERVICE_STATUS_CHECKS (
    service_name,
    service_category,
    status,
    response_time_ms,
    error_message,
    metadata
) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetLatestStatusCheck :one
SELECT * FROM SERVICE_STATUS_CHECKS
WHERE service_name = ?
ORDER BY checked_at DESC
LIMIT 1;

-- name: GetLatestStatusChecks :many
SELECT * FROM (
    SELECT *,
           ROW_NUMBER() OVER (PARTITION BY service_name ORDER BY checked_at DESC) as rn
    FROM SERVICE_STATUS_CHECKS
    WHERE checked_at >= DATE_SUB(NOW(), INTERVAL 5 MINUTE)
) t
WHERE rn = 1;

-- name: GetStatusChecksByPeriod :many
SELECT * FROM SERVICE_STATUS_CHECKS
WHERE service_name = ?
  AND checked_at BETWEEN ? AND ?
ORDER BY checked_at DESC;

-- name: DeleteOldStatusChecks :exec
DELETE FROM SERVICE_STATUS_CHECKS
WHERE checked_at < ?;

-- name: GetDailyUptimeData :many
SELECT
    date,
    uptime_percentage,
    avg_response_time_ms,
    p95_response_time_ms,
    incident_count
FROM SERVICE_UPTIME_DAILY
WHERE service_name = ?
  AND date BETWEEN ? AND ?
ORDER BY date DESC;

-- name: UpsertDailyUptime :exec
INSERT INTO SERVICE_UPTIME_DAILY (
    service_name,
    date,
    total_checks,
    successful_checks,
    uptime_percentage,
    avg_response_time_ms,
    p95_response_time_ms,
    downtime_minutes,
    incident_count
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    total_checks = VALUES(total_checks),
    successful_checks = VALUES(successful_checks),
    uptime_percentage = VALUES(uptime_percentage),
    avg_response_time_ms = VALUES(avg_response_time_ms),
    p95_response_time_ms = VALUES(p95_response_time_ms),
    downtime_minutes = VALUES(downtime_minutes),
    incident_count = VALUES(incident_count),
    updated_at = CURRENT_TIMESTAMP;

-- name: GetUptimeStats :one
SELECT
    COUNT(*) as total_checks,
    SUM(CASE WHEN status = 'operational' THEN 1 ELSE 0 END) as successful_checks,
    CAST(AVG(CASE WHEN response_time_ms IS NOT NULL THEN response_time_ms ELSE NULL END) AS UNSIGNED) as avg_response_time_ms
FROM SERVICE_STATUS_CHECKS
WHERE service_name = ?
  AND checked_at >= DATE_SUB(NOW(), INTERVAL ? HOUR);
