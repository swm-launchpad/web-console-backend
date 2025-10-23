-- Rollback: Remove needs_build column from CONTAINERS table

ALTER TABLE `CONTAINERS`
DROP COLUMN `needs_build`;
