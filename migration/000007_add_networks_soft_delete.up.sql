-- Add soft delete columns to NETWORKS table for proper FQDN management
-- This allows tracking which project owns a FQDN even after network deletion

-- Add soft delete columns
ALTER TABLE NETWORKS
ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL;

-- Add index for efficient soft delete queries
CREATE INDEX idx_networks_is_deleted ON NETWORKS(is_deleted);

-- Add comment explaining the soft delete strategy
ALTER TABLE NETWORKS COMMENT = 'Container network configurations with soft delete support for FQDN ownership tracking';
