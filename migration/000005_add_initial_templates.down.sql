-- Rollback Initial Templates
-- Version: 1.0.0
-- Description: Remove all initial templates

DELETE FROM TEMPLATES WHERE name IN (
    'Vue.js',
    'Express.js',
    'NestJS',
    'Go Gin',
    'MySQL',
    'PostgreSQL'
);
