-- Update name field length from 255 to 32 characters
-- Version: 1.0.0
-- Description: Reduce name field length for better UX and data consistency

-- Update PROJECTS table
ALTER TABLE `PROJECTS`
    MODIFY COLUMN `name` VARCHAR(32) NOT NULL;

-- Update CONTAINERS table
ALTER TABLE `CONTAINERS`
    MODIFY COLUMN `name` VARCHAR(32) NOT NULL;

-- Update VOLUMES table
ALTER TABLE `VOLUMES`
    MODIFY COLUMN `name` VARCHAR(32) NOT NULL;
