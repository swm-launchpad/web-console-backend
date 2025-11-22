-- Add webhook support to CONTAINERS table
ALTER TABLE `CONTAINERS`
ADD COLUMN `webhook_token` VARCHAR(64) NULL UNIQUE COMMENT 'Webhook authentication token for auto-deployment',
ADD COLUMN `webhook_enabled` BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Auto-deployment via webhook enabled',
ADD INDEX `idx_webhook_token` (`webhook_token`);

-- Add trigger information to DEPLOYMENTS table
ALTER TABLE `DEPLOYMENTS`
ADD COLUMN `trigger_source` VARCHAR(50) NULL COMMENT 'Deployment trigger source: manual, webhook, api',
ADD COLUMN `trigger_metadata` JSON NULL COMMENT 'Additional trigger information (container info, user, ip, etc)',
ADD INDEX `idx_trigger_source` (`trigger_source`);

-- LP-504
