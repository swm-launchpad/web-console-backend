package application

import (
	"context"
	"fmt"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	projectrepo "github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"go.uber.org/zap"
)

type GetNodePortInput struct {
	ContainerID uint
	UserID      uint
}

type GetNodePortOutput struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

type GetNodePortUseCase struct {
	containerRepo        repository.ContainerRepository
	projectRepo          projectrepo.ProjectRepository
	permissionSvc        service.PermissionService
	tektonNodePortClient infrastructure.TektonNodePortClient
	logger               logger.Logger
}

func NewGetNodePortUseCase(
	containerRepo repository.ContainerRepository,
	projectRepo projectrepo.ProjectRepository,
	permissionSvc service.PermissionService,
	tektonNodePortClient infrastructure.TektonNodePortClient,
	log logger.Logger,
) *GetNodePortUseCase {
	return &GetNodePortUseCase{
		containerRepo:        containerRepo,
		projectRepo:          projectRepo,
		permissionSvc:        permissionSvc,
		tektonNodePortClient: tektonNodePortClient,
		logger:               log,
	}
}

func (uc *GetNodePortUseCase) Execute(ctx context.Context, input GetNodePortInput) (*GetNodePortOutput, error) {
	uc.logger.Info(ctx, "get nodeport started",
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

	network := networks[0]

	// Check if tekton_event_id exists
	tektonEventID := network.TektonEventID()
	if tektonEventID == nil {
		// No NodePort has been created
		uc.logger.Info(ctx, "nodeport not created - tekton_event_id is null")
		return nil, containererrors.ErrNodePortNotFound
	}

	// tekton_event_id exists, check if we have result in DB
	externalIP := network.ExternalIP()
	externalPort := network.ExternalPort()
	expiresAt := network.ExpiresAt()

	if externalIP != nil && externalPort != nil && expiresAt != nil {
		// We have result in DB, return it
		uc.logger.Info(ctx, "nodeport info found in database",
			zap.String("external_ip", *externalIP),
			zap.Uint16("external_port", *externalPort),
		)

		return &GetNodePortOutput{
			Host:      *externalIP,
			Port:      int(*externalPort),
			Status:    "active",
			ExpiresAt: expiresAt.Format("2006-01-02T15:04:05Z07:00"),
			CreatedAt: network.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		}, nil
	}

	// No result in DB, query PipelineRun result
	project, err := uc.projectRepo.FindByID(ctx, container.ProjectID())
	if err != nil {
		uc.logger.Error(ctx, "failed to find project",
			zap.Error(err),
			zap.Uint("project_id", container.ProjectID()),
		)
		return nil, err
	}

	serviceName := project.Slug().String()
	namespace := "application"

	uc.logger.Info(ctx, "querying pipelinerun result",
		zap.String("service_name", serviceName),
		zap.String("namespace", namespace),
		zap.String("tekton_event_id", *tektonEventID),
	)

	// Query PipelineRun result by event ID
	nodeportInfo, err := uc.tektonNodePortClient.GetPipelineRunResult(ctx, *tektonEventID)
	if err != nil {
		uc.logger.Error(ctx, "failed to get pipelinerun result",
			zap.Error(err),
			zap.String("service_name", serviceName),
		)
		return nil, err
	}

	if nodeportInfo == nil {
		// PipelineRun is still running, return creating status
		uc.logger.Info(ctx, "pipelinerun still running - nodeport creating")
		return &GetNodePortOutput{
			Host:      "",
			Port:      0,
			Status:    "creating",
			ExpiresAt: "",
			CreatedAt: network.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		}, nil
	}

	// Check if PipelineRun failed
	if nodeportInfo.Status == "failed" {
		uc.logger.Warn(ctx, "pipelinerun failed, saving NULL values to database")

		// Save NULL values (empty strings and 0) to DB with expires_at = now()
		err = uc.containerRepo.UpdateNetworkNodePortResult(
			ctx,
			network.NetworkID(),
			"",                     // NULL host
			0,                      // NULL port
			nodeportInfo.ExpiresAt, // now()
		)
		if err != nil {
			uc.logger.Error(ctx, "failed to save failed nodeport result to database",
				zap.Error(err),
				zap.Uint("network_id", network.NetworkID()),
			)
			// Don't return error, just log it and return the result
		}

		return &GetNodePortOutput{
			Host:      "",
			Port:      0,
			Status:    "failed",
			ExpiresAt: nodeportInfo.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
			CreatedAt: nodeportInfo.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}, nil
	}

	// PipelineRun has completed successfully, save result to DB
	uc.logger.Info(ctx, "pipelinerun completed successfully, saving result to database",
		zap.String("host", nodeportInfo.Host),
		zap.Int("port", nodeportInfo.Port),
	)

	err = uc.containerRepo.UpdateNetworkNodePortResult(
		ctx,
		network.NetworkID(),
		nodeportInfo.Host,
		uint16(nodeportInfo.Port),
		nodeportInfo.ExpiresAt,
	)
	if err != nil {
		uc.logger.Error(ctx, "failed to save nodeport result to database",
			zap.Error(err),
			zap.Uint("network_id", network.NetworkID()),
		)
		// Don't return error, just log it and return the result
	}

	uc.logger.Info(ctx, "get nodeport completed",
		zap.String("host", nodeportInfo.Host),
		zap.Int("port", nodeportInfo.Port),
	)

	return &GetNodePortOutput{
		Host:      nodeportInfo.Host,
		Port:      nodeportInfo.Port,
		Status:    "active",
		ExpiresAt: nodeportInfo.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		CreatedAt: nodeportInfo.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// GetConnectionString returns a formatted connection string (host:port)
func (o *GetNodePortOutput) GetConnectionString() string {
	return fmt.Sprintf("%s:%d", o.Host, o.Port)
}
