-- Rollback verification_tokens table and SECRETS value column
-- Version: 1.1.0

ALTER TABLE `SECRETS` DROP COLUMN IF EXISTS `value`;

DROP TABLE IF EXISTS `verification_tokens`;
