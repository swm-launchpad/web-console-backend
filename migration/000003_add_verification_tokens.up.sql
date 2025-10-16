-- Add verification_tokens table for email verification and password reset
-- Add value column to SECRETS table
-- Version: 1.1.0
-- Description: Add support for email verification, password reset, and secret values

CREATE TABLE `verification_tokens` (
    `token_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id` INT UNSIGNED NOT NULL,
    `token` VARCHAR(255) NOT NULL,
    `token_type` ENUM('email_verification', 'password_reset') NOT NULL,
    `expires_at` TIMESTAMP NOT NULL,
    `used_at` TIMESTAMP NULL DEFAULT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`token_id`),
    UNIQUE KEY `uk_verification_tokens_token` (`token`),
    INDEX `idx_verification_tokens_user_type` (`user_id`, `token_type`),
    CONSTRAINT `fk_verification_tokens_user`
        FOREIGN KEY (`user_id`) REFERENCES `USERS`(`user_id`)
        ON DELETE CASCADE
);

-- Add value column to SECRETS table to store encrypted secret values
ALTER TABLE `SECRETS` ADD COLUMN `value` TEXT NULL AFTER `key`;
