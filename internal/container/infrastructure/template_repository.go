package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/template"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/template/value"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure/sqlc"
	"go.uber.org/zap"
)

type templateRepository struct {
	db      sqlc.DBTX
	queries *sqlc.Queries
	logger  logger.Logger
}

// NewTemplateRepository creates a new TemplateRepository instance
func NewTemplateRepository(db sqlc.DBTX, log logger.Logger) repository.TemplateRepository {
	return &templateRepository{
		db:      db,
		queries: sqlc.New(db),
		logger:  log,
	}
}

// FindAll retrieves all templates (including inactive ones)
func (r *templateRepository) FindAll(ctx context.Context) ([]*model.Template, error) {
	r.logger.Info(ctx, "template repository find all started")

	rows, err := r.queries.FindAllTemplates(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Info(ctx, "template repository find all completed (no templates found)")
			return []*model.Template{}, nil
		}
		r.logger.Error(ctx, "template repository find all failed",
			zap.Error(err),
		)
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

	r.logger.Info(ctx, "template repository find all completed",
		zap.Int("count", len(templates)),
	)
	return templates, nil
}

// FindByID retrieves a template by its ID
func (r *templateRepository) FindByID(ctx context.Context, templateID uint) (*model.Template, error) {
	r.logger.Info(ctx, "template repository find by id started",
		zap.Uint("template_id", templateID),
	)

	row, err := r.queries.FindTemplateByID(ctx, uint32(templateID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Error(ctx, "template not found",
				zap.Uint("template_id", templateID),
				zap.Error(containererrors.ErrTemplateNotFound),
			)
			return nil, containererrors.ErrTemplateNotFound
		}
		r.logger.Error(ctx, "template repository find by id failed",
			zap.Uint("template_id", templateID),
			zap.Error(err),
		)
		return nil, containererrors.ErrDatabaseOperation
	}

	template, err := r.rowToTemplate(row)
	if err != nil {
		return nil, err
	}

	r.logger.Info(ctx, "template repository find by id completed",
		zap.Uint("template_id", template.TemplateID()),
		zap.String("name", template.Name()),
	)
	return template, nil
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
