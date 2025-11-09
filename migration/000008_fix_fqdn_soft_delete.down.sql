-- Rollback: Fix FQDN soft delete
-- Note: Cannot restore original FQDNs as they were not saved
-- This is a destructive operation and down migration is a no-op

-- No action needed for rollback as we cannot restore deleted FQDNs
