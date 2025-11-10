-- Fix FQDN soft delete to prevent unique constraint violations
-- Clear FQDN on soft-deleted networks to allow reuse

-- Update existing soft-deleted networks to clear their FQDNs
-- This allows the same FQDN to be reused after deletion
UPDATE NETWORKS
SET fqdn = NULL
WHERE is_deleted = TRUE
  AND fqdn IS NOT NULL;
