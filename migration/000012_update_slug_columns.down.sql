-- Revert slug column sizes back to VARCHAR(255)

-- WARNING: This rollback will also clear data since the new format is incompatible with old format
-- Delete in order to respect foreign key constraints
-- Note: Using DELETE only if table exists to handle cases where earlier migrations were not run
-- This makes the migration idempotent

-- Delete container-related data first
DELETE FROM `MOUNTS` WHERE 1=1;
DELETE FROM `SECRETS` WHERE 1=1;
DELETE FROM `NETWORKS` WHERE 1=1;
DELETE FROM `ENV_VARS` WHERE 1=1;

-- Delete build history if exists (added in migration 000006)
SET @table_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'BUILD_HISTORY');
SET @sql = IF(@table_exists > 0, 'DELETE FROM `BUILD_HISTORY`', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Delete deployment locks if exists (added in migration 000002)
SET @table_exists = (SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'DEPLOYMENT_LOCKS');
SET @sql = IF(@table_exists > 0, 'DELETE FROM `DEPLOYMENT_LOCKS`', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Delete deployments
DELETE FROM `DEPLOYMENTS` WHERE 1=1;

-- Delete containers (they reference projects)
DELETE FROM `CONTAINERS` WHERE 1=1;

-- Delete volumes (they reference projects)
DELETE FROM `VOLUMES` WHERE 1=1;

-- Delete project users
DELETE FROM `PROJECT_USER` WHERE 1=1;

-- Delete projects
DELETE FROM `PROJECTS` WHERE 1=1;

-- Revert PROJECTS table slug column
ALTER TABLE `PROJECTS` MODIFY COLUMN `slug` VARCHAR(255) NOT NULL;

-- Revert CONTAINERS table slug column
ALTER TABLE `CONTAINERS` MODIFY COLUMN `slug` VARCHAR(255) NOT NULL;
