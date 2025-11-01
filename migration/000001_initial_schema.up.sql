-- Web Console Database Schema - Consolidated
-- Version: 2.0.0
-- Description: Consolidated database schema for container management platform
-- This file consolidates migrations 1-20 without template data

-- Drop tables in reverse dependency order (respecting FK constraints)
-- Note: PROJECTS has circular FK with DEPLOYMENTS, so we drop the FK first
SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS `PROJECT_USER`;
DROP TABLE IF EXISTS `OAUTH_STATES`;
DROP TABLE IF EXISTS `verification_tokens`;
DROP TABLE IF EXISTS `BUILD_HISTORY`;
DROP TABLE IF EXISTS `BUILD_VARS`;
DROP TABLE IF EXISTS `SECRETS`;
DROP TABLE IF EXISTS `NETWORKS`;
DROP TABLE IF EXISTS `ENV_VARS`;
DROP TABLE IF EXISTS `MOUNTS`;
DROP TABLE IF EXISTS `CONTAINERS`;
DROP TABLE IF EXISTS `GITHUB_INSTALLATIONS`;
DROP TABLE IF EXISTS `VOLUMES`;
DROP TABLE IF EXISTS `DEPLOYMENTS`;
DROP TABLE IF EXISTS `PROJECTS`;
DROP TABLE IF EXISTS `TEMPLATES`;
DROP TABLE IF EXISTS `SYSTEM_SETTINGS`;
DROP TABLE IF EXISTS `USERS`;

SET FOREIGN_KEY_CHECKS = 1;

-- ============================================================================
-- Core Tables
-- ============================================================================

-- Users table
CREATE TABLE `USERS` (
    `user_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `username` VARCHAR(100) NOT NULL,
    `password_hash` VARCHAR(255) NOT NULL,
    `password_updated_at` TIMESTAMP NULL DEFAULT NULL,
    `name` VARCHAR(100) NULL,
    `email` VARCHAR(255) NOT NULL,
    `phone` VARCHAR(20) NULL,
    `status` ENUM('active', 'inactive', 'suspended', 'pending') NOT NULL DEFAULT 'pending',
    `organization` VARCHAR(255) NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` TIMESTAMP NULL DEFAULT NULL,
    `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (`user_id`),
    UNIQUE KEY `uk_users_username` (`username`),
    UNIQUE KEY `uk_users_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci AUTO_INCREMENT=1000;

-- Projects table
CREATE TABLE `PROJECTS` (
    `project_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(255) NOT NULL,
    `slug` VARCHAR(23) NOT NULL,
    `fqdn` VARCHAR(255) NULL,
    `status` ENUM('active', 'inactive', 'suspended', 'pending') NOT NULL DEFAULT 'pending',
    `project_operation_status` ENUM('nothing', 'building', 'deploying') NOT NULL DEFAULT 'nothing'
        COMMENT 'Current operation status of the project',
    `active_deployment_id` INT UNSIGNED NULL
        COMMENT 'ID of the deployment that currently owns the deploying status',
    `plan` VARCHAR(50) NULL,
    `cpu_limit` INT UNSIGNED NULL,
    `memory_limit` INT UNSIGNED NULL,
    `disk_limit` INT UNSIGNED NULL,
    `traffic_limit` BIGINT UNSIGNED NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` TIMESTAMP NULL DEFAULT NULL,
    `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (`project_id`),
    UNIQUE KEY `uk_projects_slug` (`slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci AUTO_INCREMENT=1000;

-- Templates table (schema only, no data)
CREATE TABLE `TEMPLATES` (
    `template_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(255) NOT NULL,
    `template_body` LONGTEXT NULL,
    `template_config` JSON NULL,
    `status` ENUM('active', 'inactive', 'deprecated') NOT NULL DEFAULT 'active',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`template_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci AUTO_INCREMENT=1000;

-- Volumes table
CREATE TABLE `VOLUMES` (
    `volume_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `project_id` INT UNSIGNED NOT NULL,
    `name` VARCHAR(255) NOT NULL,
    `slug` VARCHAR(23) NULL,
    `capacity` INT UNSIGNED NOT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`volume_id`),
    UNIQUE KEY `uk_volumes_slug` (`slug`),
    INDEX `idx_volumes_project_id` (`project_id`),
    CONSTRAINT `fk_volumes_project` FOREIGN KEY (`project_id`)
        REFERENCES `PROJECTS` (`project_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci AUTO_INCREMENT=1000;

-- GitHub Installations table
CREATE TABLE `GITHUB_INSTALLATIONS` (
    `installation_id` BIGINT UNSIGNED NOT NULL,
    `user_id` INT UNSIGNED NOT NULL,
    `account_login` VARCHAR(255) NOT NULL,
    `account_type` ENUM('User', 'Organization') NOT NULL DEFAULT 'User',
    `status` ENUM('active', 'revoked') NOT NULL DEFAULT 'active',
    `cached_token` TEXT NULL,
    `token_expires_at` TIMESTAMP NULL DEFAULT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` TIMESTAMP NULL DEFAULT NULL,
    `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (`installation_id`),
    INDEX `idx_github_installations_user_id` (`user_id`),
    INDEX `idx_github_installations_account_login` (`account_login`),
    INDEX `idx_github_installations_status` (`status`),
    CONSTRAINT `fk_github_installations_user` FOREIGN KEY (`user_id`)
        REFERENCES `USERS` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Containers table
CREATE TABLE `CONTAINERS` (
    `container_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `project_id` INT UNSIGNED NOT NULL,
    `template_id` INT UNSIGNED NULL,
    `name` VARCHAR(255) NOT NULL,
    `slug` VARCHAR(23) NOT NULL,
    `stable_window` INT UNSIGNED NULL,
    `template_config` JSON NULL,
    `github_installation_id` BIGINT UNSIGNED NULL,
    `git_repository_url` VARCHAR(500) NULL,
    `git_branch` VARCHAR(100) NULL DEFAULT 'main',
    `git_commit_hash` CHAR(40) NULL,
    `git_directory_path` VARCHAR(255) NULL,
    `last_built_git_commit_hash` CHAR(40) NULL,
    `needs_build` BOOLEAN NOT NULL DEFAULT TRUE
        COMMENT 'Indicates whether build is required (set to true when build parameters change)',
    `cpu_limit` INT UNSIGNED NULL,
    `memory_limit` INT UNSIGNED NULL,
    `monthly_build_time` INT UNSIGNED NULL,
    `monthly_build_count` INT UNSIGNED NULL DEFAULT 0,
    `monthly_uptime` DECIMAL(5,2) NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` TIMESTAMP NULL DEFAULT NULL,
    `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (`container_id`),
    UNIQUE KEY `uk_containers_slug` (`slug`),
    INDEX `idx_containers_project_id` (`project_id`),
    INDEX `idx_containers_template_id` (`template_id`),
    INDEX `idx_containers_github_installation_id` (`github_installation_id`),
    CONSTRAINT `fk_containers_project` FOREIGN KEY (`project_id`)
        REFERENCES `PROJECTS` (`project_id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_containers_template` FOREIGN KEY (`template_id`)
        REFERENCES `TEMPLATES` (`template_id`) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT `fk_containers_github_installation` FOREIGN KEY (`github_installation_id`)
        REFERENCES `GITHUB_INSTALLATIONS` (`installation_id`) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci AUTO_INCREMENT=1000;

-- Environment Variables table
CREATE TABLE `ENV_VARS` (
    `env_var_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `container_id` INT UNSIGNED NOT NULL,
    `key` VARCHAR(255) NOT NULL,
    `value` TEXT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`env_var_id`),
    UNIQUE KEY `uk_env_vars_container_key` (`container_id`, `key`),
    CONSTRAINT `fk_env_vars_container` FOREIGN KEY (`container_id`)
        REFERENCES `CONTAINERS` (`container_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci AUTO_INCREMENT=1000;

-- Build Variables table
CREATE TABLE `BUILD_VARS` (
    `build_var_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `container_id` INT UNSIGNED NOT NULL,
    `key` VARCHAR(255) NOT NULL,
    `value` TEXT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`build_var_id`),
    UNIQUE KEY `uk_build_vars_container_key` (`container_id`, `key`),
    INDEX `idx_build_vars_container_id` (`container_id`),
    CONSTRAINT `fk_build_vars_container` FOREIGN KEY (`container_id`)
        REFERENCES `CONTAINERS` (`container_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci AUTO_INCREMENT=1000;

-- Secrets table
CREATE TABLE `SECRETS` (
    `secret_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `container_id` INT UNSIGNED NOT NULL,
    `key` VARCHAR(255) NOT NULL,
    `value` TEXT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`secret_id`),
    UNIQUE KEY `uk_secrets_container_key` (`container_id`, `key`),
    CONSTRAINT `fk_secrets_container` FOREIGN KEY (`container_id`)
        REFERENCES `CONTAINERS` (`container_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci AUTO_INCREMENT=1000;

-- Networks table
CREATE TABLE `NETWORKS` (
    `network_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `container_id` INT UNSIGNED NOT NULL,
    `external_ip` VARCHAR(45) NULL,
    `fqdn` VARCHAR(255) NULL,
    `external_port` SMALLINT UNSIGNED NULL,
    `internal_port` SMALLINT UNSIGNED NULL,
    `type` ENUM('tcp', 'udp', 'http') NOT NULL DEFAULT 'tcp',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`network_id`),
    INDEX `idx_networks_container_id` (`container_id`),
    CONSTRAINT `fk_networks_container` FOREIGN KEY (`container_id`)
        REFERENCES `CONTAINERS` (`container_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci AUTO_INCREMENT=1000;

-- Deployments table
CREATE TABLE `DEPLOYMENTS` (
    `deployment_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `project_id` INT UNSIGNED NOT NULL,
    `status` ENUM(
        'untracked',
        'backend_trigger_failed',
        'backend_tracking_failed',
        'backend_tracking_lost',
        'running',
        'success',
        'failed',
        'cancelled'
    ) NOT NULL DEFAULT 'untracked',
    `summary` TEXT NULL,
    `tekton_event_id` VARCHAR(255) NULL COMMENT 'Tekton event ID from API response',
    `tekton_pipeline_run_name` VARCHAR(255) NULL COMMENT 'Tekton PipelineRun name',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `started_at` TIMESTAMP NULL DEFAULT NULL,
    `finished_at` TIMESTAMP NULL DEFAULT NULL,
    PRIMARY KEY (`deployment_id`),
    INDEX `idx_deployments_project_id` (`project_id`),
    CONSTRAINT `fk_deployments_project` FOREIGN KEY (`project_id`)
        REFERENCES `PROJECTS` (`project_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci AUTO_INCREMENT=1000;

-- Build History table
CREATE TABLE `BUILD_HISTORY` (
    `build_history_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `container_id` INT UNSIGNED NOT NULL,
    `status` ENUM(
        'untracked',
        'backend_trigger_failed',
        'backend_tracking_failed',
        'backend_tracking_lost',
        'running',
        'success',
        'failed',
        'cancelled',
        'skipped'
    ) NOT NULL DEFAULT 'untracked',
    `summary` TEXT NULL,
    `tekton_event_id` VARCHAR(255) NULL COMMENT 'Tekton event ID from API response',
    `tekton_pipeline_run_name` VARCHAR(255) NULL COMMENT 'Tekton PipelineRun name',
    `git_commit_hash` CHAR(40) NULL COMMENT 'Latest commit hash from Tekton build result',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `started_at` TIMESTAMP NULL DEFAULT NULL,
    `finished_at` TIMESTAMP NULL DEFAULT NULL,
    PRIMARY KEY (`build_history_id`),
    INDEX `idx_build_history_container_id` (`container_id`),
    INDEX `idx_build_history_container_created` (`container_id`, `created_at` DESC),
    CONSTRAINT `fk_build_history_container` FOREIGN KEY (`container_id`)
        REFERENCES `CONTAINERS` (`container_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci AUTO_INCREMENT=1000;

-- Mounts table (junction table)
CREATE TABLE `MOUNTS` (
    `container_id` INT UNSIGNED NOT NULL,
    `volume_id` INT UNSIGNED NOT NULL,
    `mount_path` VARCHAR(255) NOT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`container_id`, `volume_id`),
    INDEX `idx_mounts_volume_id` (`volume_id`),
    CONSTRAINT `fk_mounts_container` FOREIGN KEY (`container_id`)
        REFERENCES `CONTAINERS` (`container_id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_mounts_volume` FOREIGN KEY (`volume_id`)
        REFERENCES `VOLUMES` (`volume_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Project User relationship table
CREATE TABLE `PROJECT_USER` (
    `project_user_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `project_id` INT UNSIGNED NOT NULL,
    `user_id` INT UNSIGNED NOT NULL,
    `role` ENUM('owner', 'member', 'guest') NOT NULL DEFAULT 'guest',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` TIMESTAMP NULL DEFAULT NULL,
    `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (`project_user_id`),
    UNIQUE KEY `uk_project_user` (`project_id`, `user_id`),
    INDEX `idx_project_user_user_id` (`user_id`),
    CONSTRAINT `fk_project_user_project` FOREIGN KEY (`project_id`)
        REFERENCES `PROJECTS` (`project_id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_project_user_user` FOREIGN KEY (`user_id`)
        REFERENCES `USERS` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci AUTO_INCREMENT=1000;

-- Verification Tokens table
CREATE TABLE `verification_tokens` (
    `token_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id` INT UNSIGNED NOT NULL,
    `token` VARCHAR(255) NOT NULL,
    `token_type` ENUM('email_verification', 'password_reset') NOT NULL,
    `expires_at` TIMESTAMP NOT NULL,
    `used_at` TIMESTAMP NULL DEFAULT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`token_id`),
    UNIQUE KEY `uk_verification_tokens_token` (`token`),
    INDEX `idx_verification_tokens_user_type` (`user_id`, `token_type`),
    CONSTRAINT `fk_verification_tokens_user`
        FOREIGN KEY (`user_id`) REFERENCES `USERS`(`user_id`)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci AUTO_INCREMENT=1000;

-- OAuth States table for CSRF protection
CREATE TABLE `OAUTH_STATES` (
    `state` VARCHAR(255) NOT NULL,
    `user_id` INT UNSIGNED NOT NULL,
    `installation_id` BIGINT UNSIGNED NULL,
    `expires_at` TIMESTAMP NOT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `consumed_at` TIMESTAMP NULL DEFAULT NULL,
    PRIMARY KEY (`state`),
    INDEX `idx_oauth_states_user_id` (`user_id`),
    INDEX `idx_oauth_states_expires_at` (`expires_at`),
    CONSTRAINT `fk_oauth_states_user` FOREIGN KEY (`user_id`)
        REFERENCES `USERS` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- System Settings table for configurable settings
CREATE TABLE `SYSTEM_SETTINGS` (
    `setting_key` VARCHAR(100) PRIMARY KEY,
    `setting_value` TEXT NOT NULL,
    `value_type` ENUM('string', 'int', 'float', 'boolean', 'json') NOT NULL,
    `category` VARCHAR(50) NOT NULL,
    `description` TEXT,
    `is_editable` BOOLEAN DEFAULT TRUE,
    `updated_by` INT UNSIGNED,
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (`updated_by`) REFERENCES `USERS`(`user_id`),
    INDEX `idx_category` (`category`),
    INDEX `idx_value_type` (`value_type`),
    INDEX `idx_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Add foreign key for active_deployment_id (must be added after DEPLOYMENTS table exists)
-- ============================================================================

ALTER TABLE `PROJECTS`
ADD CONSTRAINT `fk_active_deployment`
FOREIGN KEY (`active_deployment_id`) REFERENCES `DEPLOYMENTS`(`deployment_id`)
ON DELETE SET NULL;

-- ============================================================================
-- Insert initial system settings (pricing and limits)
-- ============================================================================

INSERT INTO `SYSTEM_SETTINGS` (`setting_key`, `setting_value`, `value_type`, `category`, `description`, `is_editable`) VALUES
-- Plan base prices (monthly, KRW)
('free_plan_base_price', '0', 'int', 'pricing', 'Free plan monthly base price (KRW)', FALSE),
('eco_plan_base_price', '1100', 'int', 'pricing', 'Eco plan monthly base price (KRW)', TRUE),
('pro_plan_base_price', '14900', 'int', 'pricing', 'Pro plan monthly base price (KRW)', TRUE),

-- Runtime pricing
('free_plan_free_minutes', '-1', 'int', 'pricing', 'Free plan free runtime minutes per month (-1 = unlimited)', FALSE),
('free_plan_runtime_price_per_minute', '0', 'float', 'pricing', 'Free plan runtime price per minute (KRW)', FALSE),
('eco_plan_free_minutes', '500', 'int', 'pricing', 'Eco plan free runtime minutes per month', TRUE),
('eco_plan_runtime_price_per_minute', '3.3', 'float', 'pricing', 'Eco plan runtime price per minute (KRW)', TRUE),
('pro_plan_free_minutes', '-1', 'int', 'pricing', 'Pro plan free runtime minutes per month (-1 = unlimited)', FALSE),
('pro_plan_runtime_price_per_minute', '0', 'float', 'pricing', 'Pro plan runtime price per minute (KRW)', FALSE),

-- Eco plan resource pricing (per minute, KRW)
('eco_cpu_price_per_core_per_minute', '30', 'float', 'pricing', 'Eco CPU pricing per core per minute (KRW)', TRUE),
('eco_memory_price_per_gb_per_minute', '15', 'float', 'pricing', 'Eco memory pricing per GB per minute (KRW)', TRUE),
('eco_disk_price_per_gb_per_month', '1000', 'int', 'pricing', 'Eco disk pricing per GB per month (KRW)', TRUE),

-- Pro plan resource pricing (per month, KRW)
('pro_cpu_price_per_core_per_month', '5000', 'int', 'pricing', 'Pro CPU pricing per core per month (KRW)', TRUE),
('pro_memory_price_per_gb_per_month', '3000', 'int', 'pricing', 'Pro memory pricing per GB per month (KRW)', TRUE),
('pro_disk_price_per_gb_per_month', '1000', 'int', 'pricing', 'Pro disk pricing per GB per month (KRW)', TRUE),

-- Free plan limits
('free_plan_cpu_limit', '500', 'int', 'limits', 'Free plan fixed CPU limit (millicores)', FALSE),
('free_plan_memory_limit', '1024', 'int', 'limits', 'Free plan fixed memory limit (Mi)', FALSE),
('free_plan_disk_limit', '1024', 'int', 'limits', 'Free plan fixed disk limit (Mi)', FALSE),
('free_plan_max_projects', '1', 'int', 'limits', 'Maximum projects per user for Free plan', TRUE),

-- Beta tier limits
('beta_tier_enabled', 'true', 'boolean', 'beta', 'Enable beta tier resource restrictions', TRUE),
('beta_tier_cpu_limit', '1000', 'int', 'beta', 'Beta tier maximum CPU limit (millicores)', TRUE),
('beta_tier_memory_limit', '2048', 'int', 'beta', 'Beta tier maximum memory limit (Mi)', TRUE),
('beta_tier_disk_limit', '3072', 'int', 'beta', 'Beta tier maximum disk limit (Mi)', TRUE);
