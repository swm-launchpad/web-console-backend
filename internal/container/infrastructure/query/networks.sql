-- Networks CRUD

-- name: CreateNetwork :execresult
INSERT INTO NETWORKS (
    container_id, internal_port, external_port, external_ip, fqdn, type,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetNetworksByContainerID :many
SELECT *
FROM NETWORKS
WHERE container_id = ?
ORDER BY internal_port ASC;

-- name: GetNetworkByID :one
SELECT *
FROM NETWORKS
WHERE network_id = ?;

-- name: UpdateNetwork :execresult
UPDATE NETWORKS SET
    external_port = ?,
    external_ip = ?,
    fqdn = ?,
    updated_at = ?
WHERE network_id = ?;

-- name: DeleteNetwork :execresult
DELETE FROM NETWORKS
WHERE network_id = ?;

-- name: DeleteNetworksByContainerID :execresult
DELETE FROM NETWORKS
WHERE container_id = ?;

-- name: CountNetworksByContainerID :one
SELECT COUNT(*) as total FROM NETWORKS WHERE container_id = ?;

-- name: CheckInternalPortExistsInProject :one
SELECT COUNT(*) > 0 as port_exists
FROM NETWORKS n
INNER JOIN CONTAINERS c ON n.container_id = c.container_id
WHERE c.project_id = ?
  AND n.internal_port = ?
  AND c.is_deleted = 0;

-- name: CheckFQDNExists :one
SELECT COUNT(*) > 0 as fqdn_exists
FROM NETWORKS
WHERE fqdn = ?;
