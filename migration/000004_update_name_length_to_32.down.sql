-- Rollback name field length from 32 to 255 characters
-- Version: 1.0.0
-- Description: Restore original name field length

-- Rollback VOLUMES table
ALTER TABLE `VOLUMES`
    MODIFY COLUMN `name` VARCHAR(255) NOT NULL;

-- Rollback CONTAINERS table
ALTER TABLE `CONTAINERS`
    MODIFY COLUMN `name` VARCHAR(255) NOT NULL;

-- Rollback PROJECTS table
ALTER TABLE `PROJECTS`
    MODIFY COLUMN `name` VARCHAR(255) NOT NULL;
