-- Rollback: Remove installation status column

ALTER TABLE `GITHUB_INSTALLATIONS`
    DROP INDEX `idx_github_installations_status`,
    DROP COLUMN `status`;
