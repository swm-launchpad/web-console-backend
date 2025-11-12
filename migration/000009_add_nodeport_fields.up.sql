-- Add NodePort tracking fields to NETWORKS table
-- These fields enable asynchronous NodePort creation tracking via Tekton pipelines

ALTER TABLE `NETWORKS`
ADD COLUMN `tekton_event_id` VARCHAR(255) NULL COMMENT 'Tekton PipelineRun name for NodePort tracking',
ADD COLUMN `expires_at` TIMESTAMP NULL COMMENT 'NodePort expiration timestamp';
