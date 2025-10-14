-- Rollback: Deployment Mechanism Refactoring
-- Version: 1.2.0
-- Description: Restore deployment locks and remove operation status from projects

-- 1. Remove added columns from DEPLOYMENTS table
ALTER TABLE `DEPLOYMENTS`
DROP COLUMN `tekton_pipeline_run_name`,
DROP COLUMN `tekton_event_id`;

-- 2. Restore original DEPLOYMENTS status enum
ALTER TABLE `DEPLOYMENTS`
MODIFY COLUMN `status` ENUM('pending', 'running', 'success', 'failed', 'cancelled') NOT NULL DEFAULT 'pending';

-- 3. Restore tekton_ref column
ALTER TABLE `DEPLOYMENTS`
ADD COLUMN `tekton_ref` VARCHAR(255) NULL AFTER `summary`;

-- 4. Remove project_operation_status from PROJECTS table
ALTER TABLE `PROJECTS`
DROP COLUMN `project_operation_status`;

-- 5. Recreate DEPLOYMENT_LOCKS table (from 000002_add_deployment_locks.up.sql)
CREATE TABLE `DEPLOYMENT_LOCKS` (
    `project_id` INT UNSIGNED NOT NULL,
    `token` BIGINT UNSIGNED NOT NULL,
    `expires_at` TIMESTAMP NOT NULL,
    PRIMARY KEY (`project_id`),
    INDEX `idx_deployment_locks_expires` (`expires_at`),
    CONSTRAINT `fk_deployment_locks_project` FOREIGN KEY (`project_id`)
        REFERENCES `PROJECTS` (`project_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
