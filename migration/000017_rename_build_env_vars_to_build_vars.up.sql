-- Rename BUILD_ENV_VARS to BUILD_VARS
-- Version: 1.4.0
-- Description: Rename BUILD_ENV_VARS table to BUILD_VARS for consistency with domain model

-- Rename the table
RENAME TABLE `BUILD_ENV_VARS` TO `BUILD_VARS`;

-- Rename the primary key column
ALTER TABLE `BUILD_VARS`
    CHANGE COLUMN `build_env_var_id` `build_var_id` INT UNSIGNED NOT NULL AUTO_INCREMENT;

-- Add index on container_id for better query performance
ALTER TABLE `BUILD_VARS`
    ADD INDEX `idx_build_vars_container_id` (`container_id`);

-- Update the unique key name to match new table name
ALTER TABLE `BUILD_VARS`
    DROP INDEX `uk_build_env_vars_container_key`,
    ADD UNIQUE KEY `uk_build_vars_container_key` (`container_id`, `key`);

-- Update the foreign key constraint name
ALTER TABLE `BUILD_VARS`
    DROP FOREIGN KEY `fk_build_env_vars_container`,
    ADD CONSTRAINT `fk_build_vars_container` FOREIGN KEY (`container_id`)
        REFERENCES `CONTAINERS` (`container_id`) ON DELETE CASCADE ON UPDATE CASCADE;

-- Add engine and charset specifications for consistency
ALTER TABLE `BUILD_VARS`
    ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
