-- Remove max_projects_per_user setting
DELETE FROM settings WHERE key_name = 'max_projects_per_user';
