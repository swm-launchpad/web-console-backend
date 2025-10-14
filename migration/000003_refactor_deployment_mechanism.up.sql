-- Deployment Mechanism Refactoring
-- Version: 1.2.0
-- Description: Remove deployment locks and add operation status to projects

-- 1. Drop DEPLOYMENT_LOCKS table
DROP TABLE IF EXISTS `DEPLOYMENT_LOCKS`;

-- 2. Add project_operation_status to PROJECTS table
ALTER TABLE `PROJECTS`
ADD COLUMN `project_operation_status` ENUM('nothing', 'building', 'deploying') NOT NULL DEFAULT 'nothing'
COMMENT 'Current operation status of the project'
AFTER `status`;

-- 3. Modify DEPLOYMENTS table
-- Drop old tekton_ref column
ALTER TABLE `DEPLOYMENTS`
DROP COLUMN `tekton_ref`;

-- Add new columns
ALTER TABLE `DEPLOYMENTS`
ADD COLUMN `tekton_event_id` VARCHAR(255) NULL COMMENT 'Tekton event ID from API response' AFTER `summary`,
ADD COLUMN `tekton_pipeline_run_name` VARCHAR(255) NULL COMMENT 'Tekton PipelineRun name' AFTER `tekton_event_id`;

-- Modify status enum
ALTER TABLE `DEPLOYMENTS`
MODIFY COLUMN `status` ENUM(
    'untracked',                    -- Initial state, not tracked yet
    'backend_trigger_failed',       -- Backend failed to trigger Tekton
    'backend_tracking_failed',      -- Backend failed to track within 5 minutes
    'backend_tracking_lost',        -- Backend lost tracking (fatal errors)
    'running',                      -- Tekton: Running
    'success',                      -- Tekton: Success
    'failed',                       -- Tekton: Failed
    'cancelled'                     -- Tekton: Cancelled (user or system initiated)
) NOT NULL DEFAULT 'untracked';
