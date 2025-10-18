-- Add slug column to VOLUMES table
-- Slug format: v{timestamp}{random} (23 characters fixed)
-- Example: v2025011812000012345678

-- Add slug column (initially NULL to allow for data migration)
ALTER TABLE `VOLUMES` ADD COLUMN `slug` VARCHAR(23) NULL AFTER `name`;

-- Add unique index on slug (will be enforced after migration)
ALTER TABLE `VOLUMES` ADD UNIQUE KEY `uk_volumes_slug` (`slug`);

-- Note: In production, you would need to:
-- 1. Generate slug values for existing rows
-- 2. Then change the column to NOT NULL
-- For new installations, the application layer will ensure slug is always set
