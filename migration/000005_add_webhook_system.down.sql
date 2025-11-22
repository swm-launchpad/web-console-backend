-- Remove trigger information from DEPLOYMENTS table
ALTER TABLE `DEPLOYMENTS`
DROP INDEX `idx_trigger_source`,
DROP COLUMN `trigger_metadata`,
DROP COLUMN `trigger_source`;

-- Remove webhook support from CONTAINERS table
ALTER TABLE `CONTAINERS`
DROP INDEX `idx_webhook_token`,
DROP COLUMN `webhook_enabled`,
DROP COLUMN `webhook_token`;

-- LP-504
