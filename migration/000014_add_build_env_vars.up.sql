-- Build Environment Variables Table
-- Version: 1.3.0
-- Description: Add BUILD_ENV_VARS table for build-time environment variables

CREATE TABLE `BUILD_ENV_VARS` (
    `build_env_var_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `container_id` INT UNSIGNED NOT NULL,
    `key` VARCHAR(255) NOT NULL,
    `value` TEXT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`build_env_var_id`),
    UNIQUE KEY `uk_build_env_vars_container_key` (`container_id`, `key`),
    CONSTRAINT `fk_build_env_vars_container` FOREIGN KEY (`container_id`)
        REFERENCES `CONTAINERS` (`container_id`) ON DELETE CASCADE ON UPDATE CASCADE
);
