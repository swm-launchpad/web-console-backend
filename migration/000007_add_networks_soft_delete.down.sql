-- Rollback: Remove soft delete columns from NETWORKS table

-- Remove index
DROP INDEX idx_networks_is_deleted ON NETWORKS;

-- Remove soft delete columns
ALTER TABLE NETWORKS DROP COLUMN deleted_at;
ALTER TABLE NETWORKS DROP COLUMN is_deleted;

-- Restore original comment
ALTER TABLE NETWORKS COMMENT = 'Container network configurations including port mappings and domain settings';
