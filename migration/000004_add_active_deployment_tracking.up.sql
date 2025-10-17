-- Active Deployment Tracking
-- Version: 1.3.0
-- Description: Add active_deployment_id to projects for tracking deployment ownership

-- Add active_deployment_id column to PROJECTS table
ALTER TABLE `PROJECTS`
ADD COLUMN `active_deployment_id` INT UNSIGNED NULL
COMMENT 'ID of the deployment that currently owns the deploying status'
AFTER `project_operation_status`;

-- Add foreign key constraint
ALTER TABLE `PROJECTS`
ADD CONSTRAINT `fk_active_deployment`
FOREIGN KEY (`active_deployment_id`) REFERENCES `DEPLOYMENTS`(`deployment_id`)
ON DELETE SET NULL;
