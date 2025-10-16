-- Template Queries
-- Templates are read-only and managed directly in the database by administrators

-- name: FindAllTemplates :many
SELECT template_id, name, template_body, template_config, status, created_at, updated_at
FROM TEMPLATES
ORDER BY template_id;

-- name: FindTemplateByID :one
SELECT template_id, name, template_body, template_config, status, created_at, updated_at
FROM TEMPLATES
WHERE template_id = ?;

-- name: FindActiveTemplates :many
SELECT template_id, name, template_body, template_config, status, created_at, updated_at
FROM TEMPLATES
WHERE status = 'active'
ORDER BY template_id;

-- name: ExistsTemplateByID :one
SELECT EXISTS(SELECT 1 FROM TEMPLATES WHERE template_id = ?) AS template_exists;
