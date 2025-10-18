-- Add GitHub App Integration Support
-- Description: Add GITHUB_INSTALLATIONS table and link to CONTAINERS

CREATE TABLE `GITHUB_INSTALLATIONS` (
    `installation_id` BIGINT UNSIGNED NOT NULL,
    `user_id` INT UNSIGNED NOT NULL,
    `account_login` VARCHAR(255) NOT NULL,
    `account_type` ENUM('User', 'Organization') NOT NULL DEFAULT 'User',
    `cached_token` TEXT NULL,
    `token_expires_at` TIMESTAMP NULL DEFAULT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` TIMESTAMP NULL DEFAULT NULL,
    `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (`installation_id`),
    INDEX `idx_github_installations_user_id` (`user_id`),
    INDEX `idx_github_installations_account_login` (`account_login`),
    CONSTRAINT `fk_github_installations_user` FOREIGN KEY (`user_id`)
        REFERENCES `USERS` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Add GitHub installation reference to CONTAINERS table
ALTER TABLE `CONTAINERS`
    ADD COLUMN `github_installation_id` BIGINT UNSIGNED NULL AFTER `template_config`,
    ADD INDEX `idx_containers_github_installation_id` (`github_installation_id`),
    ADD CONSTRAINT `fk_containers_github_installation` FOREIGN KEY (`github_installation_id`)
        REFERENCES `GITHUB_INSTALLATIONS` (`installation_id`) ON DELETE SET NULL ON UPDATE CASCADE;
