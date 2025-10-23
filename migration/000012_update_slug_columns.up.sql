-- Update slug column sizes to fixed 23 characters
-- This migration changes the slug format from name-based variable length to timestamp-based fixed length
-- Format: {prefix}{timestamp}{random} where prefix is 'p' for projects, 'c' for containers, 'v' for volumes

-- Update PROJECTS table slug column
ALTER TABLE `PROJECTS` MODIFY COLUMN `slug` VARCHAR(23) NOT NULL;

-- Update CONTAINERS table slug column
ALTER TABLE `CONTAINERS` MODIFY COLUMN `slug` VARCHAR(23) NOT NULL;

-- Note: VOLUMES table already has the correct slug size (VARCHAR(23)) from migration 000008_add_volume_slug.up.sql
