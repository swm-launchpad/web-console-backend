package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"go.uber.org/zap"
)

type UpdateNetworkInput struct {
	ContainerID  uint
	UserID       uint
	NetworkID    uint
	InternalPort *uint16
	NetworkType  string
	FQDN         *string
}

type UpdateNetworkOutput struct {
	ContainerID  uint    `json:"container_id"`
	NetworkID    uint    `json:"network_id"`
	InternalPort *uint16 `json:"internal_port,omitempty"`
	NetworkType  string  `json:"network_type"`
	FQDN         *string `json:"fqdn,omitempty"`
	UpdatedAt    string  `json:"updated_at"`
}

type UpdateNetworkUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
	txManager     db.TxManager
	logger        logger.Logger
}

func NewUpdateNetworkUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
	log logger.Logger,
) *UpdateNetworkUseCase {
	return &UpdateNetworkUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
		logger:        log,
	}
}

func (uc *UpdateNetworkUseCase) Execute(ctx context.Context, input UpdateNetworkInput) (*UpdateNetworkOutput, error) {
	uc.logger.Info(ctx, "update network started",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("user_id", input.UserID),
		zap.Uint("network_id", input.NetworkID),
	)

	var networkID uint
	var internalPort *uint16
	var networkType string
	var fqdn *string
	var updatedAt string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Check permission
		if err := uc.permissionSvc.CanUserModifyContainer(txCtx, input.UserID, input.ContainerID); err != nil {
			uc.logger.Warn(ctx, "permission check failed",
				zap.Error(err),
				zap.Uint("user_id", input.UserID),
				zap.Uint("container_id", input.ContainerID),
			)
			return err
		}

		// Get container with lock
		container, err := uc.containerRepo.FindByIDForUpdate(txCtx, input.ContainerID)
		if err != nil {
			uc.logger.Error(ctx, "failed to find container for update",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
			)
			return err
		}

		// Validate internal port uniqueness in project if internal port is provided and being changed
		if input.InternalPort != nil {
			portExists, err := uc.containerRepo.CheckInternalPortExistsInProjectExcludingSelf(
				txCtx,
				container.ProjectID(),
				*input.InternalPort,
				input.NetworkID,
			)
			if err != nil {
				uc.logger.Error(ctx, "failed to check internal port existence excluding self",
					zap.Error(err),
					zap.Uint("project_id", container.ProjectID()),
					zap.Uint16("internal_port", *input.InternalPort),
					zap.Uint("network_id", input.NetworkID),
				)
				return err
			}
			if portExists {
				uc.logger.Warn(ctx, "internal port already exists in project",
					zap.Uint("project_id", container.ProjectID()),
					zap.Uint16("internal_port", *input.InternalPort),
				)
				return containererrors.ErrDuplicateInternalPort
			}
		}

		// Validate FQDN uniqueness if FQDN is provided
		// FQDN ownership is project-scoped: check if used by OTHER projects
		if input.FQDN != nil && *input.FQDN != "" {
			fqdnExists, err := uc.containerRepo.CheckFQDNExistsInOtherProjectExcludingSelf(
				txCtx, *input.FQDN, input.NetworkID, container.ProjectID())
			if err != nil {
				uc.logger.Error(ctx, "failed to check FQDN existence in other project excluding self",
					zap.Error(err),
					zap.String("fqdn", *input.FQDN),
					zap.Uint("network_id", input.NetworkID),
					zap.Uint("project_id", container.ProjectID()),
				)
				return err
			}
			if fqdnExists {
				uc.logger.Warn(ctx, "FQDN already exists in another project",
					zap.String("fqdn", *input.FQDN),
					zap.Uint("project_id", container.ProjectID()),
				)
				return containererrors.ErrDuplicateFQDN
			}
		}

		// Create network type value object if provided
		var netType value.NetworkType
		if input.NetworkType != "" {
			netType, err = value.NewNetworkType(input.NetworkType)
			if err != nil {
				uc.logger.Error(ctx, "failed to create network type",
					zap.Error(err),
					zap.String("network_type", input.NetworkType),
				)
				return err
			}
		}

		// Update network
		network, err := container.UpdateNetwork(input.NetworkID, input.InternalPort, netType, input.FQDN)
		if err != nil {
			uc.logger.Error(ctx, "failed to update network",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
				zap.Uint("network_id", input.NetworkID),
			)
			return err
		}

		// Save container
		if err := uc.containerRepo.Save(txCtx, container); err != nil {
			uc.logger.Error(ctx, "failed to save container",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
			)
			return err
		}

		// Extract values
		networkID = network.NetworkID()
		internalPort = network.InternalPort()
		networkType = network.NetworkType().String()
		fqdn = network.FQDN()
		updatedAt = network.UpdatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "update network completed",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("network_id", networkID),
		zap.String("network_type", networkType),
	)

	return &UpdateNetworkOutput{
		ContainerID:  input.ContainerID,
		NetworkID:    networkID,
		InternalPort: internalPort,
		NetworkType:  networkType,
		FQDN:         fqdn,
		UpdatedAt:    updatedAt,
	}, nil
}
