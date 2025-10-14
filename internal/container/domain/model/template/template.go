package model

import (
	"time"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/template/value"
)

// Template represents a container template aggregate root
// Templates are read-only in the application and managed directly in the database by administrators
type Template struct {
	templateID     uint
	name           string
	templateBody   *string               // Dockerfile template (nullable)
	templateConfig *value.TemplateConfig // JSON configuration (nullable)
	status         value.TemplateStatus
	createdAt      time.Time
	updatedAt      time.Time
}

// NewTemplate creates a new Template aggregate root
// This is primarily for testing purposes as templates are managed in the database
func NewTemplate(name string, templateBody *string, templateConfig *value.TemplateConfig, status value.TemplateStatus) (*Template, error) {
	if name == "" {
		return nil, containererrors.ErrInvalidTemplateConfig
	}

	if !status.IsValid() {
		return nil, containererrors.ErrInvalidTemplateConfig
	}

	now := time.Now()
	return &Template{
		templateID:     0, // Will be set by repository
		name:           name,
		templateBody:   templateBody,
		templateConfig: templateConfig,
		status:         status,
		createdAt:      now,
		updatedAt:      time.Time{}, // Zero time for new templates (NULL in database)
	}, nil
}

// ReconstructTemplate reconstructs a template from persistence
// This is used when loading a template from the database
func ReconstructTemplate(
	templateID uint,
	name string,
	templateBody *string,
	templateConfig *value.TemplateConfig,
	status value.TemplateStatus,
	createdAt time.Time,
	updatedAt time.Time,
) *Template {
	return &Template{
		templateID:     templateID,
		name:           name,
		templateBody:   templateBody,
		templateConfig: templateConfig,
		status:         status,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}

// TemplateID returns the template ID
func (t *Template) TemplateID() uint {
	return t.templateID
}

// Name returns the template name
func (t *Template) Name() string {
	return t.name
}

// TemplateBody returns the template body (Dockerfile template)
func (t *Template) TemplateBody() *string {
	return t.templateBody
}

// TemplateConfig returns the template configuration
func (t *Template) TemplateConfig() *value.TemplateConfig {
	return t.templateConfig
}

// Status returns the template status
func (t *Template) Status() value.TemplateStatus {
	return t.status
}

// CreatedAt returns the creation timestamp
func (t *Template) CreatedAt() time.Time {
	return t.createdAt
}

// UpdatedAt returns the last update timestamp
func (t *Template) UpdatedAt() time.Time {
	return t.updatedAt
}

// IsActive checks if the template is active
func (t *Template) IsActive() bool {
	return t.status.IsActive()
}

// SetTemplateID sets the template ID (typically set by repository after persistence)
func (t *Template) SetTemplateID(id uint) {
	t.templateID = id
}
