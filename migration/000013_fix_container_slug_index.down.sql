-- Revert container slug index changes
-- This restores the previous composite unique index

-- Drop the globally unique slug index
ALTER TABLE `CONTAINERS` DROP INDEX `uk_containers_slug`;

-- Drop the project_id index we added
ALTER TABLE `CONTAINERS` DROP INDEX `idx_containers_project_id`;

-- Restore the original composite unique index
ALTER TABLE `CONTAINERS` ADD UNIQUE KEY `uk_containers_project_slug` (`project_id`, `slug`);
