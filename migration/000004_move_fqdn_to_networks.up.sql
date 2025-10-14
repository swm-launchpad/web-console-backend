-- Move FQDN from CONTAINERS to NETWORKS
-- This allows multiple domain names per container in the future

-- Step 1: Add fqdn column to NETWORKS table
ALTER TABLE `NETWORKS`
ADD COLUMN `fqdn` VARCHAR(255) NULL AFTER `external_ip`;

-- Step 2: Migrate existing FQDN data from CONTAINERS to NETWORKS
-- For each container with fqdn, set it to the first network (ordered by network_id)
UPDATE `NETWORKS` n
INNER JOIN (
    SELECT
        c.container_id,
        c.fqdn,
        MIN(n2.network_id) as first_network_id
    FROM `CONTAINERS` c
    INNER JOIN `NETWORKS` n2 ON c.container_id = n2.container_id
    WHERE c.fqdn IS NOT NULL AND c.fqdn != ''
    GROUP BY c.container_id, c.fqdn
) AS container_fqdns ON n.network_id = container_fqdns.first_network_id
SET n.fqdn = container_fqdns.fqdn;

-- Step 3: Remove fqdn column from CONTAINERS table
ALTER TABLE `CONTAINERS`
DROP COLUMN `fqdn`;
