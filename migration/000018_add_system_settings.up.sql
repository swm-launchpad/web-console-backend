-- Create SYSTEM_SETTINGS table for configurable settings
CREATE TABLE SYSTEM_SETTINGS (
    setting_key VARCHAR(100) PRIMARY KEY,
    setting_value TEXT NOT NULL,
    value_type ENUM('string', 'int', 'float', 'boolean', 'json') NOT NULL,
    category VARCHAR(50) NOT NULL,
    description TEXT,
    is_editable BOOLEAN DEFAULT TRUE,
    updated_by INT UNSIGNED,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (updated_by) REFERENCES USERS(user_id),
    INDEX idx_category (category),
    INDEX idx_value_type (value_type),
    INDEX idx_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Insert initial pricing settings
INSERT INTO SYSTEM_SETTINGS (setting_key, setting_value, value_type, category, description, is_editable) VALUES
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
