-- Rollback GitHub App Integration Support

-- Remove GitHub installation reference from CONTAINERS table
ALTER TABLE `CONTAINERS`
    DROP FOREIGN KEY `fk_containers_github_installation`,
    DROP INDEX `idx_containers_github_installation_id`,
    DROP COLUMN `github_installation_id`;

-- Drop GITHUB_INSTALLATIONS table
DROP TABLE IF EXISTS `GITHUB_INSTALLATIONS`;
