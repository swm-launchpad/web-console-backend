-- Rollback Migration: Restore FQDN to PROJECTS and remove UNIQUE constraint from NETWORKS.fqdn
-- Version: 1.1.0
-- Description:
--   Rollback changes to restore the original schema

-- Step 1: Remove UNIQUE constraint from NETWORKS.fqdn
ALTER TABLE `NETWORKS`
DROP INDEX `uk_networks_fqdn`;

-- Step 2: Add FQDN column back to PROJECTS table
-- Note: This will add the column back as NULL for all existing rows
ALTER TABLE `PROJECTS`
ADD COLUMN `fqdn` VARCHAR(255) NULL AFTER `slug`;
