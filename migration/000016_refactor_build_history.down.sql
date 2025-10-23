-- Rollback: Restore original BUILD_HISTORY table structure

-- 1. Restore git_commit_hash to original position (before modifying status)
ALTER TABLE `BUILD_HISTORY`
MODIFY COLUMN `git_commit_hash` CHAR(40) NULL
AFTER `tekton_ref`;

-- 2. Restore original status enum
ALTER TABLE `BUILD_HISTORY`
MODIFY COLUMN `status` ENUM('pending', 'running', 'success', 'failed', 'cancelled') NOT NULL DEFAULT 'pending';

-- 3. Drop new Tekton tracking columns
ALTER TABLE `BUILD_HISTORY`
DROP COLUMN `tekton_pipeline_run_name`,
DROP COLUMN `tekton_event_id`;

-- 4. Restore tekton_ref column
ALTER TABLE `BUILD_HISTORY`
ADD COLUMN `tekton_ref` VARCHAR(255) NULL AFTER `summary`;
