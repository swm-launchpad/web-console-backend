-- Migration: Update USERS.nickname column length from VARCHAR(100) to VARCHAR(32)
-- Reason: Align with name field length limit for consistency and prevent excessively long nicknames

ALTER TABLE USERS MODIFY COLUMN nickname VARCHAR(32) NOT NULL
  COMMENT 'User nickname (display name), max 32 characters';
