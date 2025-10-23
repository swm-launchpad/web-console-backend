-- Fix container slug index to enforce global uniqueness
-- Previously, the index was (project_id, slug) which only enforced uniqueness within a project
-- Now that container slugs are globally unique, we need to update the index

-- Step 1: Drop foreign key that depends on the composite index
-- MySQL requires foreign keys to have indexes on the referenced columns
-- We need to drop the FK before we can modify the indexes
ALTER TABLE `CONTAINERS` DROP FOREIGN KEY `fk_containers_project`;

-- Step 2: Drop the old composite unique index
ALTER TABLE `CONTAINERS` DROP INDEX `uk_containers_project_slug`;

-- Step 3: Add new unique index on slug only (globally unique)
ALTER TABLE `CONTAINERS` ADD UNIQUE KEY `uk_containers_slug` (`slug`);

-- Step 4: Add regular index on project_id for foreign key performance
-- (previously covered by the composite index)
ALTER TABLE `CONTAINERS` ADD INDEX `idx_containers_project_id` (`project_id`);

-- Step 5: Re-create the foreign key constraint
ALTER TABLE `CONTAINERS`
    ADD CONSTRAINT `fk_containers_project`
    FOREIGN KEY (`project_id`)
    REFERENCES `PROJECTS` (`project_id`)
    ON DELETE CASCADE
    ON UPDATE CASCADE;
