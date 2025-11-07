-- Migration: Change login from username to email, rename name to nickname
-- Step 1: Make name NOT NULL by setting default value for NULL entries
UPDATE USERS SET name = SUBSTRING_INDEX(email, '@', 1) WHERE name IS NULL OR name = '';

-- Step 2: Rename name column to nickname and make it NOT NULL
ALTER TABLE USERS CHANGE COLUMN name nickname VARCHAR(100) NOT NULL;

-- Step 3: Remove username unique index
ALTER TABLE USERS DROP INDEX uk_users_username;

-- Step 4: Drop username column
ALTER TABLE USERS DROP COLUMN username;
