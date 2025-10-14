package repository

import (
	"context"

	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/template"
)

// TemplateRepository defines the interface for template data persistence
// Templates are read-only in the application and managed directly in the database
type TemplateRepository interface {
	// FindAll retrieves all templates (including inactive ones)
	FindAll(ctx context.Context) ([]*model.Template, error)

	// FindByID retrieves a template by its ID
	FindByID(ctx context.Context, templateID uint) (*model.Template, error)

	// FindActiveTemplates retrieves only active templates
	// This is the primary method used by the application
	FindActiveTemplates(ctx context.Context) ([]*model.Template, error)

	// ExistsByID checks if a template with the given ID exists
	// Used for validating template references in containers
	ExistsByID(ctx context.Context, templateID uint) (bool, error)
}
