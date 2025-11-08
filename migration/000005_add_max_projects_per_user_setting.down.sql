-- Remove max_projects_per_user setting
DELETE FROM SYSTEM_SETTINGS WHERE setting_key = 'max_projects_per_user';
