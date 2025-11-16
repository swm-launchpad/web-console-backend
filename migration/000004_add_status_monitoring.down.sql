-- Remove status monitoring settings from SYSTEM_SETTINGS table
DELETE FROM `SYSTEM_SETTINGS` WHERE `setting_key` IN (
    'status_check_interval_seconds',
    'status_check_timeout_ms',
    'status_retention_days'
);

-- Drop SERVICE_INCIDENTS table
DROP TABLE IF EXISTS `SERVICE_INCIDENTS`;

-- Drop SERVICE_UPTIME_DAILY table
DROP TABLE IF EXISTS `SERVICE_UPTIME_DAILY`;

-- Drop SERVICE_STATUS_CHECKS table
DROP TABLE IF EXISTS `SERVICE_STATUS_CHECKS`;
