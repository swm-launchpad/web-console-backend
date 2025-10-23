-- Revert BUILD_VARS to BUILD_ENV_VARS
-- Version: 1.4.0
-- Description: Revert BUILD_VARS table back to BUILD_ENV_VARS

-- Remove engine and charset specifications
ALTER TABLE `BUILD_VARS`
    ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Revert the foreign key constraint name
ALTER TABLE `BUILD_VARS`
    DROP FOREIGN KEY `fk_build_vars_container`,
    ADD CONSTRAINT `fk_build_env_vars_container` FOREIGN KEY (`container_id`)
        REFERENCES `CONTAINERS` (`container_id`) ON DELETE CASCADE ON UPDATE CASCADE;

-- Revert the unique key name
ALTER TABLE `BUILD_VARS`
    DROP INDEX `uk_build_vars_container_key`,
    ADD UNIQUE KEY `uk_build_env_vars_container_key` (`container_id`, `key`);

-- Remove the added index
ALTER TABLE `BUILD_VARS`
    DROP INDEX `idx_build_vars_container_id`;

-- Revert the primary key column name
ALTER TABLE `BUILD_VARS`
    CHANGE COLUMN `build_var_id` `build_env_var_id` INT UNSIGNED NOT NULL AUTO_INCREMENT;

-- Revert the table name
RENAME TABLE `BUILD_VARS` TO `BUILD_ENV_VARS`;
