package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
	templatevalue "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/template/value"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	projectservice "github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	userrepository "github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"go.uber.org/zap"
)

type VolumeToCreate struct {
	Name      string
	Capacity  uint32
	MountPath string
}

type NetworkToCreate struct {
	InternalPort uint16
	NetworkType  string
	FQDN         *string
}

type CreateContainerInput struct {
	ProjectID            uint
	UserID               uint
	TemplateID           *uint
	Name                 string
	GitURL               string
	GitBranch            string
	GitSubpath           *string
	GitHubInstallationID *int64 // Optional: for private repositories
	CPULimit             uint32
	MemoryLimit          uint32
	TemplateConfig       map[string]interface{}
	Volumes              []VolumeToCreate
	Networks             []NetworkToCreate // User-specified networks (takes priority over template default_ports)
}

type CreateContainerOutput struct {
	ContainerID uint   `json:"container_id"`
	ProjectID   uint   `json:"project_id"`
	TemplateID  uint   `json:"template_id,omitempty"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	GitURL      string `json:"git_url"`
	GitBranch   string `json:"git_branch"`
	GitSubpath  string `json:"git_subpath,omitempty"`
	CPULimit    uint32 `json:"cpu_limit"`
	MemoryLimit uint32 `json:"memory_limit"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type CreateContainerUseCase struct {
	containerService      service.ContainerService
	containerRepo         repository.ContainerRepository
	templateRepo          repository.TemplateRepository
	permissionSvc         service.PermissionService
	resourceValidationSvc service.ResourceValidationService
	volumeService         projectservice.VolumeService
	installationRepo      userrepository.GitHubInstallationRepository
	txManager             db.TxManager
	logger                logger.Logger
}

func NewCreateContainerUseCase(
	containerService service.ContainerService,
	containerRepo repository.ContainerRepository,
	templateRepo repository.TemplateRepository,
	permissionSvc service.PermissionService,
	resourceValidationSvc service.ResourceValidationService,
	volumeService projectservice.VolumeService,
	installationRepo userrepository.GitHubInstallationRepository,
	txManager db.TxManager,
	log logger.Logger,
) *CreateContainerUseCase {
	return &CreateContainerUseCase{
		containerService:      containerService,
		containerRepo:         containerRepo,
		templateRepo:          templateRepo,
		permissionSvc:         permissionSvc,
		resourceValidationSvc: resourceValidationSvc,
		volumeService:         volumeService,
		installationRepo:      installationRepo,
		txManager:             txManager,
		logger:                log,
	}
}

func (uc *CreateContainerUseCase) Execute(ctx context.Context, input CreateContainerInput) (*CreateContainerOutput, error) {
	uc.logger.Info(ctx, "create container started",
		zap.Uint("project_id", input.ProjectID),
		zap.Uint("user_id", input.UserID),
		zap.String("name", input.Name),
		zap.Uint32("cpu_limit", input.CPULimit),
		zap.Uint32("memory_limit", input.MemoryLimit),
	)

	var containerID, projectID uint
	var templateID *uint
	var name, slug, gitURL, gitBranch string
	var gitDirectory *string
	var cpuLimit, memoryLimit uint32
	var createdAt, updatedAt string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Check permission
		if err := uc.permissionSvc.CanUserCreateContainer(txCtx, input.UserID, input.ProjectID); err != nil {
			uc.logger.Warn(ctx, "permission check failed",
				zap.Error(err),
				zap.Uint("user_id", input.UserID),
				zap.Uint("project_id", input.ProjectID),
			)
			return err
		}

		// Validate GitHub installation ownership if provided
		if input.GitHubInstallationID != nil {
			if err := uc.installationRepo.ValidateUserOwnership(txCtx, *input.GitHubInstallationID, input.UserID); err != nil {
				uc.logger.Error(ctx, "GitHub installation ownership validation failed",
					zap.Error(err),
					zap.Int64("installation_id", *input.GitHubInstallationID),
				)
				return err
			}
		}

		// Validate project resource limits
		if err := uc.resourceValidationSvc.ValidateProjectResourceLimits(
			txCtx,
			input.ProjectID,
			input.CPULimit,
			input.MemoryLimit,
			0, // excludeContainerID = 0 for new containers
		); err != nil {
			uc.logger.Error(ctx, "resource limits validation failed",
				zap.Error(err),
				zap.Uint("project_id", input.ProjectID),
			)
			return err
		}

		// Create Git configuration
		gitConfig, err := value.NewGitConfig(input.GitURL, input.GitBranch, input.GitSubpath)
		if err != nil {
			uc.logger.Error(ctx, "failed to create git config",
				zap.Error(err),
			)
			return err
		}

		// Create resource limits
		cpuLimitPtr := &input.CPULimit
		memLimitPtr := &input.MemoryLimit
		resourceLimits, err := value.NewResourceLimits(cpuLimitPtr, memLimitPtr)
		if err != nil {
			uc.logger.Error(ctx, "failed to create resource limits",
				zap.Error(err),
			)
			return err
		}

		// Validate Git URL if template requires git (must check before creating container)
		if input.TemplateID != nil {
			template, err := uc.templateRepo.FindByID(txCtx, *input.TemplateID)
			if err != nil {
				uc.logger.Error(ctx, "failed to fetch template for validation",
					zap.Error(err),
					zap.Uint("template_id", *input.TemplateID),
				)
				return err
			}

			templateConfig := template.TemplateConfig()
			if templateConfig != nil && templateConfig.GetRequiresGit() && input.GitURL == "" {
				uc.logger.Warn(ctx, "template requires git but no git url provided",
					zap.Uint("template_id", *input.TemplateID),
					zap.String("template_name", template.Name()),
				)
				return containererrors.ErrGitURLRequired
			}
		}

		// Create container through service
		// Slug is automatically generated from name by the service
		container, err := uc.containerService.CreateContainer(
			txCtx,
			input.ProjectID,
			input.Name,
			gitConfig,
			resourceLimits,
			input.TemplateID,
			input.TemplateConfig,
			input.GitHubInstallationID,
		)
		if err != nil {
			uc.logger.Error(ctx, "failed to create container",
				zap.Error(err),
				zap.String("name", input.Name),
			)
			return err
		}

		// Create volumes and add mounts if volumes are specified
		if len(input.Volumes) > 0 {
			for _, volToCreate := range input.Volumes {
				// Create volume using VolumeService
				volume, err := uc.volumeService.CreateVolume(
					txCtx,
					input.ProjectID,
					volToCreate.Name,
					volToCreate.Capacity,
				)
				if err != nil {
					return err
				}

				// Add mount to container
				_, err = container.AddMount(volume.VolumeID(), volToCreate.MountPath)
				if err != nil {
					return err
				}
			}

			// Save container with mounts using repository
			if err := uc.containerRepo.Save(txCtx, container); err != nil {
				return err
			}
		}

		// Network creation - Priority: User-specified networks > Template default_ports
		// User-specified networks (from frontend form)
		if len(input.Networks) > 0 {
			uc.logger.Info(ctx, "creating networks from user input",
				zap.Int("network_count", len(input.Networks)),
			)

			for _, networkInput := range input.Networks {
				// Validate internal port uniqueness in project
				portExists, err := uc.containerRepo.CheckInternalPortExistsInProject(
					txCtx,
					input.ProjectID,
					networkInput.InternalPort,
				)
				if err != nil {
					uc.logger.Error(ctx, "failed to check internal port existence",
						zap.Error(err),
						zap.Uint("project_id", input.ProjectID),
						zap.Uint16("internal_port", networkInput.InternalPort),
					)
					return err
				}
				if portExists {
					uc.logger.Warn(ctx, "internal port already exists in project",
						zap.Uint("project_id", input.ProjectID),
						zap.Uint16("internal_port", networkInput.InternalPort),
					)
					return containererrors.ErrDuplicateInternalPort
				}

				// Validate FQDN uniqueness if provided
				// Check project-scoped FQDN ownership with soft-delete consideration
				if networkInput.FQDN != nil && *networkInput.FQDN != "" {
					fqdnExists, err := uc.containerRepo.CheckFQDNExistsForProject(txCtx, *networkInput.FQDN, input.ProjectID)
					if err != nil {
						uc.logger.Error(ctx, "failed to check FQDN existence for project",
							zap.Error(err),
							zap.String("fqdn", *networkInput.FQDN),
							zap.Uint("project_id", input.ProjectID),
						)
						return err
					}
					if fqdnExists {
						uc.logger.Warn(ctx, "FQDN already exists (duplicate in project or owned by active container in other project)",
							zap.String("fqdn", *networkInput.FQDN),
							zap.Uint("project_id", input.ProjectID),
						)
						return containererrors.ErrDuplicateFQDN
					}
				}

				// Parse network type
				networkType, err := value.NewNetworkType(networkInput.NetworkType)
				if err != nil {
					uc.logger.Error(ctx, "invalid network type in user input",
						zap.Error(err),
						zap.Uint16("internal_port", networkInput.InternalPort),
						zap.String("network_type", networkInput.NetworkType),
					)
					return err
				}

				// Add network to container
				internalPort := networkInput.InternalPort
				_, err = container.AddNetwork(
					&internalPort,
					nil, // external_port not supported in user input
					networkType,
					nil, // external_ip not supported in user input
					networkInput.FQDN,
				)
				if err != nil {
					uc.logger.Error(ctx, "failed to add network from user input",
						zap.Error(err),
						zap.Uint16("internal_port", networkInput.InternalPort),
						zap.String("network_type", networkInput.NetworkType),
					)
					return err
				}

				uc.logger.Info(ctx, "network created from user input",
					zap.Uint16("internal_port", networkInput.InternalPort),
					zap.String("network_type", networkInput.NetworkType),
				)
			}

			// Save container with user-specified networks
			if err := uc.containerRepo.Save(txCtx, container); err != nil {
				return err
			}

			uc.logger.Info(ctx, "container saved with user-specified networks",
				zap.Int("network_count", len(input.Networks)),
			)
		} else if input.TemplateID != nil {
			// Auto-create networks from template default_ports (fallback)
			uc.logger.Info(ctx, "fetching template for network auto-creation",
				zap.Uint("template_id", *input.TemplateID),
			)

			template, err := uc.templateRepo.FindByID(txCtx, *input.TemplateID)
			if err != nil {
				uc.logger.Error(ctx, "failed to fetch template for network creation",
					zap.Error(err),
					zap.Uint("template_id", *input.TemplateID),
				)
				return err
			}

			// Get template config
			templateConfig := template.TemplateConfig()
			if templateConfig == nil {
				uc.logger.Warn(ctx, "template has no config",
					zap.Uint("template_id", *input.TemplateID),
				)
				templateConfig = &templatevalue.TemplateConfig{}
			}

			uc.logger.Info(ctx, "template fetched successfully",
				zap.Uint("template_id", *input.TemplateID),
				zap.String("template_name", template.Name()),
				zap.Bool("requires_git", templateConfig.GetRequiresGit()),
				zap.Int("default_ports_count", len(templateConfig.DefaultPorts)),
			)

			// Create networks from default_ports
			defaultPorts := templateConfig.DefaultPorts
			if len(defaultPorts) > 0 {
				for _, defaultPort := range defaultPorts {
					// Validate internal port uniqueness in project
					// Containers in same project share K8s pod network interface
					portExists, err := uc.containerRepo.CheckInternalPortExistsInProject(
						txCtx,
						input.ProjectID,
						defaultPort.InternalPort,
					)
					if err != nil {
						uc.logger.Error(ctx, "failed to check internal port existence",
							zap.Error(err),
							zap.Uint("project_id", input.ProjectID),
							zap.Uint16("internal_port", defaultPort.InternalPort),
						)
						return err
					}
					if portExists {
						uc.logger.Warn(ctx, "internal port already exists in project",
							zap.Uint("project_id", input.ProjectID),
							zap.Uint16("internal_port", defaultPort.InternalPort),
						)
						return containererrors.ErrDuplicateInternalPort
					}

					// Parse network type
					networkType, err := value.NewNetworkType(defaultPort.NetworkType)
					if err != nil {
						uc.logger.Error(ctx, "invalid network type in default_ports",
							zap.Error(err),
							zap.Uint16("internal_port", defaultPort.InternalPort),
							zap.String("network_type", defaultPort.NetworkType),
						)
						return err
					}

					// Add network to container
					internalPort := defaultPort.InternalPort
					_, err = container.AddNetwork(
						&internalPort,
						defaultPort.ExternalPort,
						networkType,
						defaultPort.ExternalIP,
						nil, // fqdn is not set in default_ports
					)
					if err != nil {
						uc.logger.Error(ctx, "failed to add network from default_ports",
							zap.Error(err),
							zap.Uint16("internal_port", defaultPort.InternalPort),
							zap.String("network_type", defaultPort.NetworkType),
						)
						return err
					}

					uc.logger.Info(ctx, "network auto-created from template default_ports",
						zap.Uint16("internal_port", defaultPort.InternalPort),
						zap.String("network_type", defaultPort.NetworkType),
					)
				}

				uc.logger.Info(ctx, "saving container with auto-created networks",
					zap.Int("network_count", len(defaultPorts)),
				)

				// Save container with networks using repository
				if err := uc.containerRepo.Save(txCtx, container); err != nil {
					return err
				}

				uc.logger.Info(ctx, "container saved successfully with networks")

				uc.logger.Info(ctx, "auto-created networks from template default_ports",
					zap.Uint("template_id", *input.TemplateID),
					zap.Int("network_count", len(defaultPorts)),
				)
			}
		}

		// Extract primitive values within transaction
		containerID = container.ContainerID()
		projectID = container.ProjectID()
		templateID = container.TemplateID()
		name = container.Name()
		slug = container.Slug().String()
		gitURL = container.GitConfig().RepositoryURL()
		gitBranch = container.GitConfig().Branch()
		gitDirectory = container.GitConfig().DirectoryPath()
		cpuLimit = container.ResourceLimits().CPULimitOrDefault(0)
		memoryLimit = container.ResourceLimits().MemoryLimitOrDefault(0)
		createdAt = container.CreatedAt().Format("2006-01-02T15:04:05Z")
		if !container.UpdatedAt().IsZero() {
			updatedAt = container.UpdatedAt().Format("2006-01-02T15:04:05Z")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "create container completed",
		zap.Uint("container_id", containerID),
		zap.Uint("project_id", projectID),
		zap.String("slug", slug),
	)

	// Build output after successful transaction
	output := &CreateContainerOutput{
		ContainerID: containerID,
		ProjectID:   projectID,
		Name:        name,
		Slug:        slug,
		GitURL:      gitURL,
		GitBranch:   gitBranch,
		CPULimit:    cpuLimit,
		MemoryLimit: memoryLimit,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	if templateID != nil {
		output.TemplateID = *templateID
	}

	if gitDirectory != nil {
		output.GitSubpath = *gitDirectory
	}

	return output, nil
}
