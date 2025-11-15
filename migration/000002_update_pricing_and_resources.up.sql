-- 000002_update_pricing_and_resources.up.sql
-- Update pricing policy and resource allocation for all plans

-- ============================================================
-- PART 1: Update SYSTEM_SETTINGS (Pricing and Resource Limits)
-- ============================================================

-- Update Disk pricing (Eco and Pro plans)
UPDATE `SYSTEM_SETTINGS`
SET `setting_value` = '200'
WHERE `setting_key` = 'eco_disk_price_per_gb_per_month';

UPDATE `SYSTEM_SETTINGS`
SET `setting_value` = '200'
WHERE `setting_key` = 'pro_disk_price_per_gb_per_month';

-- Update Free plan resource limits
UPDATE `SYSTEM_SETTINGS`
SET `setting_value` = '1000'
WHERE `setting_key` = 'free_plan_cpu_limit';

UPDATE `SYSTEM_SETTINGS`
SET `setting_value` = '2048'
WHERE `setting_key` = 'free_plan_memory_limit';

-- Update Beta tier limits
UPDATE `SYSTEM_SETTINGS`
SET `setting_value` = '2000'
WHERE `setting_key` = 'beta_tier_cpu_limit';

UPDATE `SYSTEM_SETTINGS`
SET `setting_value` = '4096'
WHERE `setting_key` = 'beta_tier_memory_limit';

UPDATE `SYSTEM_SETTINGS`
SET `setting_value` = '10240'
WHERE `setting_key` = 'beta_tier_disk_limit';

-- ============================================================
-- PART 2: Update existing PROJECTS resources
-- ============================================================

-- Update Free plan projects: CPU -> 1000m, RAM -> 2048Mi (Disk stays 1024Mi)
UPDATE `PROJECTS`
SET
    `cpu_limit` = 1000,
    `memory_limit` = 2048
WHERE `plan` = 'free';

-- Update Eco plan projects: CPU -> 1000m, RAM -> 2048Mi (Disk stays as is or 2048Mi if less)
UPDATE `PROJECTS`
SET
    `cpu_limit` = 1000,
    `memory_limit` = 2048,
    `disk_limit` = GREATEST(`disk_limit`, 2048)
WHERE `plan` = 'eco';

-- Update Pro plan projects: CPU -> 1000m, RAM -> 2048Mi, Disk -> 10240Mi
UPDATE `PROJECTS`
SET
    `cpu_limit` = 1000,
    `memory_limit` = 2048,
    `disk_limit` = GREATEST(`disk_limit`, 10240)
WHERE `plan` = 'pro';
