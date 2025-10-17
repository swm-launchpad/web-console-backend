-- Rollback Active Deployment Tracking
-- Version: 1.3.0

-- Drop foreign key constraint
ALTER TABLE `PROJECTS`
DROP FOREIGN KEY `fk_active_deployment`;

-- Drop active_deployment_id column
ALTER TABLE `PROJECTS`
DROP COLUMN `active_deployment_id`;
