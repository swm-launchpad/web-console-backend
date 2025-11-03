-- Migration: Remove FQDN from PROJECTS and add UNIQUE constraint to NETWORKS.fqdn
-- Version: 1.1.0
-- Description:
--   1. Remove fqdn column from PROJECTS table (no longer needed at project level)
--   2. Add UNIQUE constraint to NETWORKS.fqdn (enforce uniqueness across all networks)

-- Step 1: Remove FQDN column from PROJECTS table
ALTER TABLE `PROJECTS` DROP COLUMN `fqdn`;

-- Step 2: Add UNIQUE constraint to NETWORKS.fqdn
-- Note: MySQL allows multiple NULL values with UNIQUE constraint
-- This ensures FQDNs are unique only when they are explicitly set
ALTER TABLE `NETWORKS`
ADD UNIQUE KEY `uk_networks_fqdn` (`fqdn`);
