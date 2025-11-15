-- Remove UNIQUE constraint on fqdn to support soft delete
-- This allows multiple soft-deleted networks to retain their FQDN values
-- Application layer enforces uniqueness with proper business rules:
-- 1. Same project: can reuse soft-deleted FQDN immediately (same deployment cycle)
-- 2. Different project: cannot reuse soft-deleted FQDN (infrastructure still using it)
-- 3. Project deletion: CASCADE DELETE releases FQDN for reuse

ALTER TABLE `NETWORKS` DROP INDEX `uk_networks_fqdn`;
