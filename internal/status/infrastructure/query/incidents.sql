-- name: CreateIncident :execresult
INSERT INTO SERVICE_INCIDENTS (
    service_name,
    severity,
    title,
    description,
    status,
    started_at,
    affected_services
) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetIncidentByID :one
SELECT * FROM SERVICE_INCIDENTS
WHERE incident_id = ?;

-- name: GetActiveIncidents :many
SELECT * FROM SERVICE_INCIDENTS
WHERE status != 'resolved'
ORDER BY started_at DESC;

-- name: GetIncidentsByService :many
SELECT * FROM SERVICE_INCIDENTS
WHERE service_name = ?
ORDER BY started_at DESC
LIMIT ?;

-- name: GetRecentIncidents :many
SELECT * FROM SERVICE_INCIDENTS
ORDER BY started_at DESC
LIMIT ?;

-- name: UpdateIncident :exec
UPDATE SERVICE_INCIDENTS
SET
    severity = ?,
    title = ?,
    description = ?,
    status = ?,
    resolved_at = ?,
    duration_minutes = ?,
    affected_services = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE incident_id = ?;

-- name: ResolveIncident :exec
UPDATE SERVICE_INCIDENTS
SET
    status = 'resolved',
    resolved_at = ?,
    duration_minutes = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE incident_id = ?;
