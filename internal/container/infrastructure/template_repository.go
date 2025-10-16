package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/template"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/template/value"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure/sqlc"
)

type templateRepository struct {
	db      sqlc.DBTX
	queries *sqlc.Queries
}

// NewTemplateRepository creates a new TemplateRepository instance
func NewTemplateRepository(db sqlc.DBTX) repository.TemplateRepository {
	return &templateRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

// FindAll retrieves all templates (including inactive ones)
func (r *templateRepository) FindAll(ctx context.Context) ([]*model.Template, error) {
	rows, err := r.queries.FindAllTemplates(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*model.Template{}, nil
		}
		return nil, containererrors.ErrDatabaseOperation
	}

	templates := make([]*model.Template, 0, len(rows))
	for _, row := range rows {
		template, err := r.rowToTemplate(row)
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}

	return templates, nil
}

// FindByID retrieves a template by its ID
func (r *templateRepository) FindByID(ctx context.Context, templateID uint) (*model.Template, error) {
	row, err := r.queries.FindTemplateByID(ctx, uint32(templateID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, containererrors.ErrTemplateNotFound
		}
		return nil, containererrors.ErrDatabaseOperation
	}

	return r.rowToTemplate(row)
}

// FindActiveTemplates retrieves only active templates
func (r *templateRepository) FindActiveTemplates(ctx context.Context) ([]*model.Template, error) {
	rows, err := r.queries.FindActiveTemplates(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*model.Template{}, nil
		}
		return nil, containererrors.ErrDatabaseOperation
	}

	templates := make([]*model.Template, 0, len(rows))
	for _, row := range rows {
		template, err := r.rowToTemplate(row)
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}

	return templates, nil
}

// ExistsByID checks if a template with the given ID exists
func (r *templateRepository) ExistsByID(ctx context.Context, templateID uint) (bool, error) {
	exists, err := r.queries.ExistsTemplateByID(ctx, uint32(templateID))
	if err != nil {
		return false, containererrors.ErrDatabaseOperation
	}

	return exists, nil
}

// rowToTemplate converts a SQLC Template row to domain Template model
func (r *templateRepository) rowToTemplate(row sqlc.Template) (*model.Template, error) {
	// Parse status
	status, err := value.NewTemplateStatus(string(row.Status))
	if err != nil {
		return nil, containererrors.ErrInvalidTemplateConfig
	}

	// Parse template_body (nullable)
	var templateBody *string
	if row.TemplateBody.Valid {
		templateBody = &row.TemplateBody.String
	}

	// Parse template_config (nullable JSON)
	var templateConfig *value.TemplateConfig
	if len(row.TemplateConfig) > 0 {
		config, err := value.NewTemplateConfig(string(row.TemplateConfig))
		if err != nil {
			return nil, containererrors.ErrInvalidTemplateConfig
		}
		templateConfig = config
	}

	// Parse updated_at
	var updatedAt time.Time
	if row.UpdatedAt.Valid {
		updatedAt = row.UpdatedAt.Time
	}

	// Reconstruct template
	template := model.ReconstructTemplate(
		uint(row.TemplateID),
		row.Name,
		templateBody,
		templateConfig,
		status,
		row.CreatedAt,
		updatedAt,
	)

	return template, nil
}
