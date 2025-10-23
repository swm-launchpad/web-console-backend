-- Refactor BUILD_HISTORY Table Structure
-- Version: 1.3.0
-- Description: Update BUILD_HISTORY to match DEPLOYMENTS structure with enhanced status tracking

-- 1. Drop old tekton_ref column
ALTER TABLE `BUILD_HISTORY`
DROP COLUMN `tekton_ref`;

-- 2. Add new Tekton tracking columns
ALTER TABLE `BUILD_HISTORY`
ADD COLUMN `tekton_event_id` VARCHAR(255) NULL COMMENT 'Tekton event ID from API response' AFTER `summary`,
ADD COLUMN `tekton_pipeline_run_name` VARCHAR(255) NULL COMMENT 'Tekton PipelineRun name' AFTER `tekton_event_id`;

-- 3. Modify status enum to match DEPLOYMENTS structure and add 'skipped' state
ALTER TABLE `BUILD_HISTORY`
MODIFY COLUMN `status` ENUM(
    'untracked',                    -- Initial state, not tracked yet
    'backend_trigger_failed',       -- Backend failed to trigger Tekton
    'backend_tracking_failed',      -- Backend failed to track within 5 minutes
    'backend_tracking_lost',        -- Backend lost tracking (fatal errors)
    'running',                      -- Tekton: Running
    'success',                      -- Tekton: Success
    'failed',                       -- Tekton: Failed
    'cancelled',                    -- Tekton: Cancelled
    'skipped'                       -- Tekton: Build skipped (no changes)
) NOT NULL DEFAULT 'untracked';

-- 4. Adjust git_commit_hash column position and add comment
ALTER TABLE `BUILD_HISTORY`
MODIFY COLUMN `git_commit_hash` CHAR(40) NULL
COMMENT 'Latest commit hash from Tekton build result'
AFTER `tekton_pipeline_run_name`;
