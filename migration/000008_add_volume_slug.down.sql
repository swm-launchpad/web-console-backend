-- Remove slug column from VOLUMES table

-- Drop unique index first
ALTER TABLE `VOLUMES` DROP INDEX `uk_volumes_slug`;

-- Drop slug column
ALTER TABLE `VOLUMES` DROP COLUMN `slug`;
