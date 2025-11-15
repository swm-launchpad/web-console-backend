-- 000002_update_pricing_and_resources.down.sql
-- Rollback pricing policy and resource allocation changes

-- ============================================================
-- PART 1: Rollback SYSTEM_SETTINGS (Pricing and Resource Limits)
-- ============================================================

-- Rollback Disk pricing (Eco and Pro plans)
UPDATE `SYSTEM_SETTINGS`
SET `setting_value` = '1000'
WHERE `setting_key` = 'eco_disk_price_per_gb_per_month';

UPDATE `SYSTEM_SETTINGS`
SET `setting_value` = '1000'
WHERE `setting_key` = 'pro_disk_price_per_gb_per_month';

-- Rollback Free plan resource limits
UPDATE `SYSTEM_SETTINGS`
SET `setting_value` = '500'
WHERE `setting_key` = 'free_plan_cpu_limit';

UPDATE `SYSTEM_SETTINGS`
SET `setting_value` = '1024'
WHERE `setting_key` = 'free_plan_memory_limit';

-- Rollback Beta tier limits
UPDATE `SYSTEM_SETTINGS`
SET `setting_value` = '1000'
WHERE `setting_key` = 'beta_tier_cpu_limit';

UPDATE `SYSTEM_SETTINGS`
SET `setting_value` = '2048'
WHERE `setting_key` = 'beta_tier_memory_limit';

UPDATE `SYSTEM_SETTINGS`
SET `setting_value` = '3072'
WHERE `setting_key` = 'beta_tier_disk_limit';

-- ============================================================
-- PART 2: Rollback existing PROJECTS resources
-- ============================================================

-- Rollback Free plan projects: CPU -> 500m, RAM -> 1024Mi
UPDATE `PROJECTS`
SET
    `cpu_limit` = 500,
    `memory_limit` = 1024
WHERE `plan` = 'free';

-- Rollback Eco plan projects: CPU -> 500m, RAM -> 1024Mi, Disk -> 2048Mi (if was changed)
UPDATE `PROJECTS`
SET
    `cpu_limit` = 500,
    `memory_limit` = 1024,
    `disk_limit` = 2048
WHERE `plan` = 'eco';

-- Rollback Pro plan projects: CPU -> 500m, RAM -> 1024Mi, Disk -> 2048Mi (if was changed)
UPDATE `PROJECTS`
SET
    `cpu_limit` = 500,
    `memory_limit` = 1024,
    `disk_limit` = 2048
WHERE `plan` = 'pro';
