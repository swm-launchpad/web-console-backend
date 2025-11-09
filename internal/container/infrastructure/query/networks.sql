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
  AND is_deleted = 0
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
UPDATE NETWORKS SET
    is_deleted = TRUE,
    deleted_at = ?
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
  AND n.is_deleted = 0
  AND c.is_deleted = 0
FOR UPDATE;

-- name: CheckFQDNExists :one
-- Check if FQDN is used anywhere in the system
-- Simple availability check across all projects
SELECT COUNT(*) > 0 as fqdn_exists
FROM NETWORKS n
INNER JOIN CONTAINERS c ON n.container_id = c.container_id
WHERE n.fqdn = ?
  AND n.is_deleted = 0
  AND c.is_deleted = 0
FOR UPDATE;

-- name: CheckFQDNExistsInOtherProject :one
-- Check if FQDN is used by another project (for AddNetwork)
-- FQDN ownership is project-scoped: once a project uses a FQDN, it's reserved for that project
-- Checks both active and deleted networks to preserve FQDN ownership
SELECT COUNT(*) > 0 as fqdn_exists
FROM NETWORKS n
INNER JOIN CONTAINERS c ON n.container_id = c.container_id
WHERE n.fqdn = ?
  AND c.project_id != ?
  AND c.is_deleted = 0
FOR UPDATE;

-- name: CheckFQDNExistsInOtherProjectExcludingSelf :one
-- Check if FQDN is used by another project, excluding self (for UpdateNetwork)
-- Allows updating a network's FQDN to the same value or reusing FQDN within same project
-- Checks both active and deleted networks to preserve FQDN ownership
SELECT COUNT(*) > 0 as fqdn_exists
FROM NETWORKS n
INNER JOIN CONTAINERS c ON n.container_id = c.container_id
WHERE n.fqdn = ?
  AND n.network_id != ?
  AND c.project_id != ?
  AND c.is_deleted = 0
FOR UPDATE;

-- name: CheckInternalPortExistsInProjectExcludingSelf :one
-- Check if internal port exists in project, excluding self (for UpdateNetwork)
-- Internal ports must be unique within a project (containers share pod network interface)
SELECT COUNT(*) > 0 as port_exists
FROM NETWORKS n
INNER JOIN CONTAINERS c ON n.container_id = c.container_id
WHERE c.project_id = ?
  AND n.internal_port = ?
  AND n.network_id != ?
  AND n.is_deleted = 0
  AND c.is_deleted = 0
FOR UPDATE;

-- name: SoftDeleteNetworksByContainerID :execresult
-- Soft delete all networks when a container is deleted
-- This preserves FQDN ownership tracking even after container deletion
UPDATE NETWORKS SET
    is_deleted = TRUE,
    deleted_at = ?
WHERE container_id = ?
  AND is_deleted = 0;

-- name: SoftDeleteNetworkByID :execresult
-- Soft delete a specific network by ID
-- Preserves FQDN ownership for the project
UPDATE NETWORKS SET
    is_deleted = TRUE,
    deleted_at = ?
WHERE network_id = ?
  AND is_deleted = 0;
