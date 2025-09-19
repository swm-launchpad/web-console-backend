-- Web Console Database Schema
-- Version: 1.0.0
-- Description: Initial database schema for container management platform

DROP TABLE IF EXISTS `PROJECT_USER`;
DROP TABLE IF EXISTS `MOUNTS`;
DROP TABLE IF EXISTS `SECRETS`;
DROP TABLE IF EXISTS `BUILD_HISTORY`;
DROP TABLE IF EXISTS `NETWORKS`;
DROP TABLE IF EXISTS `ENV_VARS`;
DROP TABLE IF EXISTS `DEPLOYMENTS`;
DROP TABLE IF EXISTS `CONTAINERS`;
DROP TABLE IF EXISTS `VOLUMES`;
DROP TABLE IF EXISTS `TEMPLATES`;
DROP TABLE IF EXISTS `PROJECTS`;
DROP TABLE IF EXISTS `USERS`;

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
);

CREATE TABLE `PROJECTS` (
    `project_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(255) NOT NULL,
    `slug` VARCHAR(255) NOT NULL,
    `fqdn` VARCHAR(255) NULL,
    `status` ENUM('active', 'inactive', 'suspended', 'pending') NOT NULL DEFAULT 'pending',
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
);

CREATE TABLE `TEMPLATES` (
    `template_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(255) NOT NULL,
    `template_body` LONGTEXT NULL,
    `template_config` JSON NULL,
    `status` ENUM('active', 'inactive', 'deprecated') NOT NULL DEFAULT 'active',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`template_id`)
);

CREATE TABLE `VOLUMES` (
    `volume_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `project_id` INT UNSIGNED NOT NULL,
    `name` VARCHAR(255) NOT NULL,
    `capacity` INT UNSIGNED NOT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`volume_id`),
    INDEX `idx_volumes_project_id` (`project_id`),
    CONSTRAINT `fk_volumes_project` FOREIGN KEY (`project_id`)
        REFERENCES `PROJECTS` (`project_id`) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE `CONTAINERS` (
    `container_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `project_id` INT UNSIGNED NOT NULL,
    `template_id` INT UNSIGNED NULL,
    `name` VARCHAR(255) NOT NULL,
    `slug` VARCHAR(255) NOT NULL,
    `fqdn` VARCHAR(255) NULL,
    `stable_window` INT UNSIGNED NULL,
    `template_config` JSON NULL,
    `git_repository_url` VARCHAR(500) NULL,
    `git_branch` VARCHAR(100) NULL DEFAULT 'main',
    `git_commit_hash` CHAR(40) NULL,
    `git_directory_path` VARCHAR(255) NULL,
    `last_built_git_commit_hash` CHAR(40) NULL,
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
    UNIQUE KEY `uk_containers_project_slug` (`project_id`, `slug`),
    INDEX `idx_containers_template_id` (`template_id`),
    CONSTRAINT `fk_containers_project` FOREIGN KEY (`project_id`)
        REFERENCES `PROJECTS` (`project_id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_containers_template` FOREIGN KEY (`template_id`)
        REFERENCES `TEMPLATES` (`template_id`) ON DELETE SET NULL ON UPDATE CASCADE
);

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
);

CREATE TABLE `DEPLOYMENTS` (
    `deployment_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `project_id` INT UNSIGNED NOT NULL,
    `status` ENUM('pending', 'running', 'success', 'failed', 'cancelled') NOT NULL DEFAULT 'pending',
    `summary` TEXT NULL,
    `tekton_ref` VARCHAR(255) NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `started_at` TIMESTAMP NULL DEFAULT NULL,
    `finished_at` TIMESTAMP NULL DEFAULT NULL,
    PRIMARY KEY (`deployment_id`),
    INDEX `idx_deployments_project_id` (`project_id`),
    CONSTRAINT `fk_deployments_project` FOREIGN KEY (`project_id`)
        REFERENCES `PROJECTS` (`project_id`) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE `NETWORKS` (
    `network_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `container_id` INT UNSIGNED NOT NULL,
    `external_ip` VARCHAR(45) NULL,
    `external_port` SMALLINT UNSIGNED NULL,
    `internal_port` SMALLINT UNSIGNED NULL,
    `type` ENUM('tcp', 'udp', 'http') NOT NULL DEFAULT 'tcp',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`network_id`),
    INDEX `idx_networks_container_id` (`container_id`),
    CONSTRAINT `fk_networks_container` FOREIGN KEY (`container_id`)
        REFERENCES `CONTAINERS` (`container_id`) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE `BUILD_HISTORY` (
    `build_history_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `container_id` INT UNSIGNED NOT NULL,
    `status` ENUM('pending', 'running', 'success', 'failed', 'cancelled') NOT NULL DEFAULT 'pending',
    `summary` TEXT NULL,
    `tekton_ref` VARCHAR(255) NULL,
    `git_commit_hash` CHAR(40) NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `started_at` TIMESTAMP NULL DEFAULT NULL,
    `finished_at` TIMESTAMP NULL DEFAULT NULL,
    PRIMARY KEY (`build_history_id`),
    INDEX `idx_build_history_container_id` (`container_id`),
    INDEX `idx_build_history_container_created` (`container_id`, `created_at` DESC),
    CONSTRAINT `fk_build_history_container` FOREIGN KEY (`container_id`)
        REFERENCES `CONTAINERS` (`container_id`) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE `SECRETS` (
    `secret_id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `container_id` INT UNSIGNED NOT NULL,
    `key` VARCHAR(255) NOT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`secret_id`),
    UNIQUE KEY `uk_secrets_container_key` (`container_id`, `key`),
    CONSTRAINT `fk_secrets_container` FOREIGN KEY (`container_id`)
        REFERENCES `CONTAINERS` (`container_id`) ON DELETE CASCADE ON UPDATE CASCADE
);

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
);

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
);