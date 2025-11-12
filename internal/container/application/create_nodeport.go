package application

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	projectrepo "github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"go.uber.org/zap"
)

type CreateNodePortInput struct {
	ContainerID uint
	UserID      uint
}

type CreateNodePortOutput struct {
	Status string `json:"status"` // "accepted" - creation started
}

type CreateNodePortUseCase struct {
	containerRepo        repository.ContainerRepository
	projectRepo          projectrepo.ProjectRepository
	permissionSvc        service.PermissionService
	tektonNodePortClient infrastructure.TektonNodePortClient
	logger               logger.Logger
}

func NewCreateNodePortUseCase(
	containerRepo repository.ContainerRepository,
	projectRepo projectrepo.ProjectRepository,
	permissionSvc service.PermissionService,
	tektonNodePortClient infrastructure.TektonNodePortClient,
	log logger.Logger,
) *CreateNodePortUseCase {
	return &CreateNodePortUseCase{
		containerRepo:        containerRepo,
		projectRepo:          projectRepo,
		permissionSvc:        permissionSvc,
		tektonNodePortClient: tektonNodePortClient,
		logger:               log,
	}
}

func (uc *CreateNodePortUseCase) Execute(ctx context.Context, input CreateNodePortInput) (*CreateNodePortOutput, error) {
	uc.logger.Info(ctx, "create nodeport started",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("user_id", input.UserID),
	)

	// Check permission
	if err := uc.permissionSvc.CanUserModifyContainer(ctx, input.UserID, input.ContainerID); err != nil {
		uc.logger.Warn(ctx, "permission check failed",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
			zap.Uint("container_id", input.ContainerID),
		)
		return nil, err
	}

	// Get container
	container, err := uc.containerRepo.FindByID(ctx, input.ContainerID)
	if err != nil {
		uc.logger.Error(ctx, "failed to find container",
			zap.Error(err),
			zap.Uint("container_id", input.ContainerID),
		)
		return nil, err
	}

	// Get container's network
	networks := container.Networks()
	if len(networks) == 0 {
		uc.logger.Error(ctx, "container has no network configured",
			zap.Uint("container_id", input.ContainerID),
		)
		return nil, containererrors.ErrNoTCPNetwork
	}

	// Since user mentioned "현재 컨테이너마다 하나의 네트워크만 존재할 수 있는 상황"
	// we can assume there's only one network
	network := networks[0]

	// Validate network type is TCP
	if network.NetworkType().String() != "tcp" {
		uc.logger.Error(ctx, "network type is not tcp",
			zap.String("network_type", network.NetworkType().String()),
		)
		return nil, containererrors.ErrNodePortNotSupported
	}

	// Get internal port (target port)
	internalPort := network.InternalPort()
	if internalPort == nil {
		uc.logger.Error(ctx, "network has no internal port configured")
		return nil, containererrors.ErrInternalPortRequired
	}

	// Check NodePort state based on tekton_event_id and expires_at
	tektonEventID := network.TektonEventID()
	expiresAt := network.ExpiresAt()

	if tektonEventID != nil {
		// tekton_event_id exists, check expires_at
		if expiresAt == nil {
			// expires_at is NULL -> NodePort is being created
			uc.logger.Warn(ctx, "nodeport is already being created",
				zap.String("tekton_event_id", *tektonEventID),
			)
			return nil, containererrors.ErrNodePortCreating
		}

		// Check if expires_at is still valid (not expired)
		if time.Now().Before(*expiresAt) {
			// NodePort is still active
			uc.logger.Warn(ctx, "nodeport is already active",
				zap.String("tekton_event_id", *tektonEventID),
				zap.Time("expires_at", *expiresAt),
			)
			return nil, containererrors.ErrNodePortAlreadyActive
		}

		// expires_at has expired, allow recreation
		uc.logger.Info(ctx, "nodeport has expired, allowing recreation",
			zap.String("tekton_event_id", *tektonEventID),
			zap.Time("expires_at", *expiresAt),
		)
	}

	// Get project to obtain Knative Service name
	project, err := uc.projectRepo.FindByID(ctx, container.ProjectID())
	if err != nil {
		uc.logger.Error(ctx, "failed to find project",
			zap.Error(err),
			zap.Uint("project_id", container.ProjectID()),
		)
		return nil, err
	}

	// Project slug is the Knative Service name
	serviceName := project.Slug().String()
	namespace := "application" // Fixed namespace for Knative services
	targetPort := int(*internalPort)
	ttlSeconds := 1800 // Fixed 30 minutes

	uc.logger.Info(ctx, "triggering nodeport creation",
		zap.String("service_name", serviceName),
		zap.String("namespace", namespace),
		zap.Int("target_port", targetPort),
		zap.Int("ttl_seconds", ttlSeconds),
	)

	// Trigger Tekton pipeline to create NodePort
	params := infrastructure.NodePortCreationParams{
		ServiceName: serviceName,
		Namespace:   namespace,
		TargetPort:  targetPort,
		TTLSeconds:  ttlSeconds,
	}

	pipelineRunName, err := uc.tektonNodePortClient.TriggerNodePortCreation(ctx, params)
	if err != nil {
		uc.logger.Error(ctx, "failed to trigger nodeport creation",
			zap.Error(err),
			zap.String("service_name", serviceName),
		)
		return nil, err
	}

	// Store tekton_event_id in database and clear NodePort fields
	err = uc.containerRepo.UpdateNetworkTektonEventID(ctx, network.NetworkID(), pipelineRunName)
	if err != nil {
		uc.logger.Error(ctx, "failed to store tekton_event_id",
			zap.Error(err),
			zap.Uint("network_id", network.NetworkID()),
			zap.String("tekton_event_id", pipelineRunName),
		)
		return nil, err
	}

	uc.logger.Info(ctx, "create nodeport accepted, pipelinerun created",
		zap.String("pipelinerun_name", pipelineRunName),
	)

	return &CreateNodePortOutput{
		Status: "accepted",
	}, nil
}
