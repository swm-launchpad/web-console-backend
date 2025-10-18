-- Create OAuth states table for CSRF protection
-- Description: Store OAuth state tokens to prevent replay attacks

CREATE TABLE `OAUTH_STATES` (
    `state` VARCHAR(255) NOT NULL,
    `user_id` INT UNSIGNED NOT NULL,
    `installation_id` BIGINT UNSIGNED NULL, -- NULL until callback, then populated for verification
    `expires_at` TIMESTAMP NOT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `consumed_at` TIMESTAMP NULL DEFAULT NULL, -- Set when state is used (one-time use)
    PRIMARY KEY (`state`),
    INDEX `idx_oauth_states_user_id` (`user_id`),
    INDEX `idx_oauth_states_expires_at` (`expires_at`),
    CONSTRAINT `fk_oauth_states_user` FOREIGN KEY (`user_id`)
        REFERENCES `USERS` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
