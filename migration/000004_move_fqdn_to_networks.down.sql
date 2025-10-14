-- Rollback: Move FQDN back from NETWORKS to CONTAINERS

-- Step 1: Add fqdn column back to CONTAINERS table
ALTER TABLE `CONTAINERS`
ADD COLUMN `fqdn` VARCHAR(255) NULL AFTER `slug`;

-- Step 2: Migrate FQDN data from NETWORKS back to CONTAINERS
-- For each container, get fqdn from the first network that has one
UPDATE `CONTAINERS` c
INNER JOIN (
    SELECT
        n.container_id,
        MIN(n.network_id) as first_network_id
    FROM `NETWORKS` n
    WHERE n.fqdn IS NOT NULL AND n.fqdn != ''
    GROUP BY n.container_id
) AS network_fqdns ON c.container_id = network_fqdns.container_id
INNER JOIN `NETWORKS` n ON n.network_id = network_fqdns.first_network_id
SET c.fqdn = n.fqdn;

-- Step 3: Remove fqdn column from NETWORKS table
ALTER TABLE `NETWORKS`
DROP COLUMN `fqdn`;
