-- Add max_projects_per_user setting for project count limit
INSERT INTO SYSTEM_SETTINGS (setting_key, setting_value, value_type, category, description, is_editable)
VALUES ('max_projects_per_user', '3', 'int', 'limits', 'Maximum projects per user (all plans)', TRUE);
