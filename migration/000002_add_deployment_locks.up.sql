-- Deployment Locks Table
-- Version: 1.1.0
-- Description: Add deployment locks table for preventing concurrent deployments

CREATE TABLE `DEPLOYMENT_LOCKS` (
    `project_id` INT UNSIGNED NOT NULL,
    `token` BIGINT UNSIGNED NOT NULL,
    `expires_at` TIMESTAMP NOT NULL,
    PRIMARY KEY (`project_id`),
    INDEX `idx_deployment_locks_expires` (`expires_at`),
    CONSTRAINT `fk_deployment_locks_project` FOREIGN KEY (`project_id`)
        REFERENCES `PROJECTS` (`project_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
