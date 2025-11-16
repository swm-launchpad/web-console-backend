-- Restore UNIQUE constraint on fqdn
-- Note: This rollback requires that all soft-deleted networks have fqdn=NULL
-- Otherwise, the UNIQUE constraint creation will fail due to duplicate values

-- First, set fqdn to NULL for all soft-deleted networks to avoid constraint violation
UPDATE `NETWORKS` SET `fqdn` = NULL WHERE `is_deleted` = 1;

-- Restore the UNIQUE constraint
ALTER TABLE `NETWORKS` ADD UNIQUE KEY `uk_networks_fqdn` (`fqdn`);
