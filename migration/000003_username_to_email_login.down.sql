-- Rollback migration: Restore username column and rename nickname back to name
-- Step 1: Add username column back
ALTER TABLE USERS ADD COLUMN username VARCHAR(100) AFTER user_id;

-- Step 2: Set username from email (take part before @)
UPDATE USERS SET username = SUBSTRING_INDEX(email, '@', 1);

-- Step 3: Make username NOT NULL
ALTER TABLE USERS MODIFY COLUMN username VARCHAR(100) NOT NULL;

-- Step 4: Add unique index back on username
ALTER TABLE USERS ADD UNIQUE INDEX uk_users_username (username);

-- Step 5: Rename nickname back to name and make it nullable
ALTER TABLE USERS CHANGE COLUMN nickname name VARCHAR(100) NULL;
