-- Project Summary Queries
-- These queries aggregate container and domain information for project list view

-- name: GetProjectsSummary :many
SELECT
    p.project_id,
    COUNT(DISTINCT c.container_id) as container_count,
    COUNT(DISTINCT CASE WHEN n.fqdn IS NOT NULL AND n.fqdn != '' THEN n.network_id END) as domain_count,
    GROUP_CONCAT(DISTINCT CASE WHEN n.fqdn IS NOT NULL AND n.fqdn != '' THEN n.fqdn END ORDER BY n.fqdn SEPARATOR ',') as domains,
    COALESCE(
        (SELECT SUM(c2.cpu_limit)
         FROM CONTAINERS c2
         WHERE c2.project_id = p.project_id AND c2.is_deleted = FALSE),
        0
    ) as total_cpu_used,
    COALESCE(
        (SELECT SUM(c2.memory_limit)
         FROM CONTAINERS c2
         WHERE c2.project_id = p.project_id AND c2.is_deleted = FALSE),
        0
    ) as total_memory_used,
    COALESCE(
        (SELECT SUM(v.capacity)
         FROM VOLUMES v
         WHERE v.project_id = p.project_id),
        0
    ) as total_disk_used
FROM PROJECTS p
LEFT JOIN CONTAINERS c ON p.project_id = c.project_id AND c.is_deleted = FALSE
LEFT JOIN NETWORKS n ON c.container_id = n.container_id AND n.is_deleted = FALSE
WHERE p.project_id IN (sqlc.slice('project_ids'))
AND p.is_deleted = FALSE
GROUP BY p.project_id
ORDER BY p.project_id;
