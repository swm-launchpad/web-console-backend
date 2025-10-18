-- Add installation status column to track active/revoked state
-- Description: Add status column to GITHUB_INSTALLATIONS table

ALTER TABLE `GITHUB_INSTALLATIONS`
    ADD COLUMN `status` ENUM('active', 'revoked') NOT NULL DEFAULT 'active' AFTER `account_type`,
    ADD INDEX `idx_github_installations_status` (`status`);

-- Update existing records to have active status (already the default)
UPDATE `GITHUB_INSTALLATIONS` SET `status` = 'active' WHERE `status` IS NULL OR `status` = '';
