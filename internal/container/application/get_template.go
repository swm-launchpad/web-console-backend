package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/template/value"
)

type GetTemplateInput struct {
	TemplateID uint
}

type GetTemplateOutput struct {
	TemplateID       uint                    `json:"template_id"`
	Name             string                  `json:"name"`
	Description      string                  `json:"description,omitempty"`
	TemplateBody     string                  `json:"template_body,omitempty"`
	TemplateOptions  []value.TemplateOption  `json:"template_options,omitempty"`
	TemplateEnv      []value.TemplateEnv     `json:"template_env,omitempty"`
	TemplatePorts    []value.TemplatePort    `json:"template_ports,omitempty"`
	TemplateVolumes  []value.TemplateVolume  `json:"template_volumes,omitempty"`
	DefaultPorts     []value.DefaultPort     `json:"default_ports,omitempty"`
	DefaultEnv       []value.DefaultEnv      `json:"default_env,omitempty"`
	DefaultVolumes   []value.DefaultVolume   `json:"default_volumes,omitempty"`
	DefaultResources *value.DefaultResources `json:"default_resources,omitempty"`
	Categories       []string                `json:"categories"`
	DisplayOrder     int                     `json:"display_order"`
	IconName         string                  `json:"icon_name,omitempty"`
	Status           string                  `json:"status"`
	RequiresGit      bool                    `json:"requires_git"`
	Version          string                  `json:"version,omitempty"`
}

type GetTemplateUseCase struct {
	templateRepo repository.TemplateRepository
}

func NewGetTemplateUseCase(templateRepo repository.TemplateRepository) *GetTemplateUseCase {
	return &GetTemplateUseCase{
		templateRepo: templateRepo,
	}
}

func (uc *GetTemplateUseCase) Execute(ctx context.Context, input GetTemplateInput) (*GetTemplateOutput, error) {
	// Get template by ID
	template, err := uc.templateRepo.FindByID(ctx, input.TemplateID)
	if err != nil {
		return nil, err
	}

	// Build output
	output := &GetTemplateOutput{
		TemplateID: template.TemplateID(),
		Name:       template.Name(),
		Status:     template.Status().String(),
	}

	// Add template body if available
	if body := template.TemplateBody(); body != nil {
		output.TemplateBody = *body
	}

	// Extract configuration fields if available
	if config := template.TemplateConfig(); config != nil {
		output.Description = config.GetDescription()
		output.Categories = config.GetCategories()
		output.DisplayOrder = config.GetDisplayOrder()
		output.IconName = config.GetIconName()
		output.RequiresGit = config.GetRequiresGit()
		output.Version = config.GetVersion()
		output.TemplateOptions = config.TemplateOptions
		output.TemplateEnv = config.TemplateEnv
		output.TemplatePorts = config.TemplatePorts
		output.TemplateVolumes = config.TemplateVolumes
		output.DefaultPorts = config.DefaultPorts
		output.DefaultEnv = config.DefaultEnv
		output.DefaultVolumes = config.DefaultVolumes
		output.DefaultResources = config.DefaultResources
	}

	return output, nil
}
