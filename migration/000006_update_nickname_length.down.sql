-- Rollback migration: Restore USERS.nickname column length from VARCHAR(32) to VARCHAR(100)

ALTER TABLE USERS MODIFY COLUMN nickname VARCHAR(100) NOT NULL
  COMMENT 'User nickname (display name)';
