-- ============================================================================
-- BREAKING CHANGE WARNING - READ CAREFULLY BEFORE APPLYING
-- ============================================================================
--
-- This migration DELETES ALL EXISTING DATA from the following tables:
--   - PROJECTS (and all related PROJECT_USER entries)
--   - CONTAINERS (and all related MOUNTS, SECRETS, NETWORKS, ENV_VARS)
--   - VOLUMES
--   - DEPLOYMENTS, BUILD_HISTORY, DEPLOYMENT_LOCKS
--
-- ONLY SAFE FOR: Development and staging environments
-- DO NOT APPLY IN PRODUCTION without implementing a proper data migration strategy
--
-- ============================================================================
--
-- Update slug column sizes to fixed 23 characters
-- This migration changes the slug format from name-based variable length to timestamp-based fixed length
-- Format: {prefix}{timestamp}{random} where prefix is 'p' for projects, 'c' for containers, 'v' for volumes
--
-- REASON FOR DATA DELETION:
-- All existing projects, containers, and volumes have incompatible slug formats
-- that cannot be automatically converted to the new timestamp-based format.
-- A production migration would need to:
--   1. Export all data with current slugs
--   2. Generate new slugs for each resource
--   3. Update all references (URLs, bookmarks, external integrations)
--   4. Re-import data with new slugs

-- Delete in order to respect foreign key constraints
-- Note: Using DELETE only if table exists to handle cases where earlier migrations were not run
-- This makes the migration idempotent

-- ============================================================================
-- SAFETY: Log deletion counts for audit trail (ALWAYS SUCCEEDS)
-- ============================================================================
-- This section logs what will be deleted but NEVER fails the migration
-- Production safety must be enforced at deployment pipeline level, not in SQL

-- Count records to be deleted
SELECT COUNT(*) INTO @projects_to_delete FROM PROJECTS;
SELECT COUNT(*) INTO @containers_to_delete FROM CONTAINERS;
SELECT COUNT(*) INTO @volumes_to_delete FROM VOLUMES;
SELECT COUNT(*) INTO @deployments_to_delete FROM DEPLOYMENTS;
SELECT COUNT(*) INTO @project_users_to_delete FROM PROJECT_USER;

-- Log deletion summary (appears in migration logs)
-- WARNING: If you see large numbers here in production, STOP IMMEDIATELY
SELECT CONCAT(
    '⚠️  MIGRATION 000012 DELETION SUMMARY ⚠️ ',
    'Projects: ', @projects_to_delete, ', ',
    'Containers: ', @containers_to_delete, ', ',
    'Volumes: ', @volumes_to_delete, ', ',
    'Deployments: ', @deployments_to_delete, ', ',
    'Project Users: ', @project_users_to_delete,
    ' | This migration DELETES ALL data - Dev/Staging ONLY'
) AS CRITICAL_WARNING;

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

-- Update PROJECTS table slug column
ALTER TABLE `PROJECTS` MODIFY COLUMN `slug` VARCHAR(23) NOT NULL;

-- Update CONTAINERS table slug column
ALTER TABLE `CONTAINERS` MODIFY COLUMN `slug` VARCHAR(23) NOT NULL;

-- Note: VOLUMES table already has the correct slug size (VARCHAR(23)) from migration 000008_add_volume_slug.up.sql
