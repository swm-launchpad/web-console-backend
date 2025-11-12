-- Remove NodePort tracking fields from NETWORKS table

ALTER TABLE `NETWORKS`
DROP COLUMN `tekton_event_id`,
DROP COLUMN `expires_at`;
