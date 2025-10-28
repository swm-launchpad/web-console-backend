package build

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

type GetContainersForBuildInput struct {
	ProjectID uint
}

type BuildContainerOutput struct {
	ContainerID         uint                   `json:"container_id"`
	Name                string                 `json:"name"`
	Slug                string                 `json:"slug"`
	TemplateID          *uint                  `json:"template_id,omitempty"`
	TemplateBody        *string                `json:"template_body,omitempty"`
	TemplateConfig      map[string]interface{} `json:"template_config"`
	GitRepositoryURL    string                 `json:"git_repository_url"`
	GitBranch           string                 `json:"git_branch"`
	GitDirectoryPath    *string                `json:"git_directory_path,omitempty"`
	LastBuiltCommitHash *string                `json:"last_built_commit_hash,omitempty"`
	NeedsBuild          bool                   `json:"needs_build"`
	BuildVars           map[string]string      `json:"build_vars"`
	InstallationID      *int64                 `json:"installation_id,omitempty"`
}

type GetContainersForBuildOutput struct {
	Containers []BuildContainerOutput `json:"containers"`
}

type GetContainersForBuildUseCase struct {
	containerService   service.ContainerService
	templateRepository repository.TemplateRepository
	logger             logger.Logger
}

func NewGetContainersForBuildUseCase(
	containerService service.ContainerService,
	templateRepository repository.TemplateRepository,
	log logger.Logger,
) *GetContainersForBuildUseCase {
	return &GetContainersForBuildUseCase{
		containerService:   containerService,
		templateRepository: templateRepository,
		logger:             log,
	}
}

func (uc *GetContainersForBuildUseCase) Execute(ctx context.Context, input GetContainersForBuildInput) (*GetContainersForBuildOutput, error) {
	// Get all containers for the project via service
	containers, err := uc.containerService.ListContainersByProjectID(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// Build output
	output := &GetContainersForBuildOutput{
		Containers: make([]BuildContainerOutput, 0, len(containers)),
	}

	for _, container := range containers {
		// Build build vars map (defensive copy to prevent aliasing)
		buildVars := make(map[string]string)
		for _, bv := range container.BuildVars() {
			buildVars[bv.Key()] = bv.Value()
		}

		// Deep copy template config to prevent snapshot aliasing
		// Maps are reference types - shallow copy would allow nested mutations
		// to affect the snapshot, defeating change detection
		templateConfig, err := deepCopyTemplateConfig(container.TemplateConfig())
		if err != nil {
			// Log serialization error for diagnostic purposes
			uc.logger.Warn(ctx, "Failed to deep copy template config, using nil",
				zap.Uint("container_id", container.ContainerID()),
				zap.Error(err),
			)
			templateConfig = nil
		}

		// Get template body if template is configured
		var templateBody *string
		if container.TemplateID() != nil {
			template, err := uc.templateRepository.FindByID(ctx, *container.TemplateID())
			if err != nil {
				// Template ID is set but template not found - this indicates data inconsistency
				// or infrastructure failure. Return error instead of silently continuing.
				return nil, err
			}
			templateBody = template.TemplateBody()
		}

		containerOutput := BuildContainerOutput{
			ContainerID:         container.ContainerID(),
			Name:                container.Name(),
			Slug:                container.Slug().String(),
			TemplateID:          container.TemplateID(),
			TemplateBody:        templateBody,
			TemplateConfig:      templateConfig,
			GitRepositoryURL:    container.GitConfig().RepositoryURL(),
			GitBranch:           container.GitConfig().Branch(),
			GitDirectoryPath:    container.GitConfig().DirectoryPath(),
			LastBuiltCommitHash: container.LastBuiltGitCommitHash(),
			NeedsBuild:          container.NeedsBuild(),
			BuildVars:           buildVars,
			InstallationID:      container.GitHubInstallationID(),
		}

		output.Containers = append(output.Containers, containerOutput)
	}

	return output, nil
}

// deepCopyTemplateConfig performs a deep copy of template config using JSON serialization
// This ensures nested maps/slices are fully cloned, preventing snapshot aliasing
// Returns error if serialization fails to help diagnose unexpected template payloads
func deepCopyTemplateConfig(src map[string]interface{}) (map[string]interface{}, error) {
	if src == nil {
		return nil, nil
	}

	// Use JSON round-trip for deep copy
	// This handles arbitrary nesting of maps, slices, and primitives
	data, err := json.Marshal(src)
	if err != nil {
		// Return error to help diagnose unexpected template payloads
		return nil, fmt.Errorf("failed to marshal template config: %w", err)
	}

	var dst map[string]interface{}
	if err := json.Unmarshal(data, &dst); err != nil {
		// Return error to help diagnose unexpected template payloads
		return nil, fmt.Errorf("failed to unmarshal template config: %w", err)
	}

	return dst, nil
}
