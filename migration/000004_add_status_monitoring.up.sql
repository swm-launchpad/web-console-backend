-- Create SERVICE_STATUS_CHECKS table for storing individual health check results
CREATE TABLE `SERVICE_STATUS_CHECKS` (
    `check_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `service_name` VARCHAR(50) NOT NULL COMMENT 'Service identifier (api_server, web_console, mysql, etc.)',
    `service_category` ENUM('core', 'build_deploy', 'infrastructure') NOT NULL,
    `status` ENUM('operational', 'degraded', 'down') NOT NULL,
    `response_time_ms` INT UNSIGNED NULL COMMENT 'Response time in milliseconds',
    `error_message` TEXT NULL COMMENT 'Error details if status is not operational',
    `checked_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `metadata` JSON NULL COMMENT 'Additional check details',
    PRIMARY KEY (`check_id`),
    INDEX `idx_service_name_checked_at` (`service_name`, `checked_at` DESC),
    INDEX `idx_checked_at` (`checked_at` DESC),
    INDEX `idx_service_category` (`service_category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Individual health check results (30-day retention)';

-- Create SERVICE_UPTIME_DAILY table for storing aggregated daily uptime statistics
CREATE TABLE `SERVICE_UPTIME_DAILY` (
    `uptime_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `service_name` VARCHAR(50) NOT NULL,
    `date` DATE NOT NULL COMMENT 'Date in UTC',
    `total_checks` INT UNSIGNED NOT NULL DEFAULT 0,
    `successful_checks` INT UNSIGNED NOT NULL DEFAULT 0,
    `uptime_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00 COMMENT '0.00-100.00',
    `avg_response_time_ms` INT UNSIGNED NULL,
    `p95_response_time_ms` INT UNSIGNED NULL,
    `downtime_minutes` INT UNSIGNED NOT NULL DEFAULT 0,
    `incident_count` SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`uptime_id`),
    UNIQUE KEY `uk_service_date` (`service_name`, `date`),
    INDEX `idx_date` (`date` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Daily uptime statistics (permanent retention)';

-- Create SERVICE_INCIDENTS table for storing incident and outage records
CREATE TABLE `SERVICE_INCIDENTS` (
    `incident_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `service_name` VARCHAR(50) NOT NULL,
    `severity` ENUM('minor', 'major', 'critical') NOT NULL,
    `title` VARCHAR(255) NOT NULL,
    `description` TEXT NULL,
    `status` ENUM('investigating', 'identified', 'monitoring', 'resolved') NOT NULL DEFAULT 'investigating',
    `started_at` TIMESTAMP NOT NULL,
    `resolved_at` TIMESTAMP NULL,
    `duration_minutes` INT UNSIGNED NULL COMMENT 'Calculated on resolution',
    `affected_services` JSON NULL COMMENT 'List of affected service names',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`incident_id`),
    INDEX `idx_service_started_at` (`service_name`, `started_at` DESC),
    INDEX `idx_status` (`status`),
    INDEX `idx_severity` (`severity`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Incident and outage records (permanent retention)';

-- Add status monitoring settings to SYSTEM_SETTINGS table
INSERT INTO `SYSTEM_SETTINGS` (`setting_key`, `setting_value`, `value_type`, `category`, `description`, `is_editable`) VALUES
('status_check_interval_seconds', '60', 'int', 'monitoring', 'Health check interval in seconds', TRUE),
('status_check_timeout_ms', '5000', 'int', 'monitoring', 'Health check timeout in milliseconds', TRUE),
('status_retention_days', '30', 'int', 'monitoring', 'Days to retain status check history', TRUE);
