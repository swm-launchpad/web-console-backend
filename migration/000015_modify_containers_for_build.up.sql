-- Modify CONTAINERS Table for Build Tracking
-- Version: 1.3.0
-- Description: Add needs_build field to CONTAINERS table
-- Note: git_commit_hash is kept for backward compatibility with existing code

-- Add needs_build column
ALTER TABLE `CONTAINERS`
ADD COLUMN `needs_build` BOOLEAN NOT NULL DEFAULT TRUE
COMMENT 'Indicates whether build is required (set to true when build parameters change)'
AFTER `last_built_git_commit_hash`;
