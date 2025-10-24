package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"go.uber.org/zap"
)

type AddNetworkInput struct {
	ContainerID  uint
	UserID       uint
	InternalPort *uint16
	ExternalPort *uint16
	NetworkType  string
	ExternalIP   *string
	FQDN         *string
}

type AddNetworkOutput struct {
	ContainerID  uint    `json:"container_id"`
	NetworkID    uint    `json:"network_id"`
	InternalPort *uint16 `json:"internal_port,omitempty"`
	ExternalPort *uint16 `json:"external_port,omitempty"`
	NetworkType  string  `json:"network_type"`
	ExternalIP   *string `json:"external_ip,omitempty"`
	FQDN         *string `json:"fqdn,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

type AddNetworkUseCase struct {
	containerRepo repository.ContainerRepository
	permissionSvc service.PermissionService
	txManager     db.TxManager
	logger        logger.Logger
}

func NewAddNetworkUseCase(
	containerRepo repository.ContainerRepository,
	permissionSvc service.PermissionService,
	txManager db.TxManager,
	log logger.Logger,
) *AddNetworkUseCase {
	return &AddNetworkUseCase{
		containerRepo: containerRepo,
		permissionSvc: permissionSvc,
		txManager:     txManager,
		logger:        log,
	}
}

func (uc *AddNetworkUseCase) Execute(ctx context.Context, input AddNetworkInput) (*AddNetworkOutput, error) {
	uc.logger.Info(ctx, "add network started",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("user_id", input.UserID),
		zap.String("network_type", input.NetworkType),
	)

	var networkID uint
	var internalPort, externalPort *uint16
	var networkType string
	var externalIP, fqdn *string
	var createdAt string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Check permission
		if err := uc.permissionSvc.CanUserModifyContainer(txCtx, input.UserID, input.ContainerID); err != nil {
			uc.logger.Error(ctx, "permission check failed",
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

		// Create network type value object
		netType, err := value.NewNetworkType(input.NetworkType)
		if err != nil {
			uc.logger.Error(ctx, "failed to create network type",
				zap.Error(err),
				zap.String("network_type", input.NetworkType),
			)
			return err
		}

		// Add network
		network, err := container.AddNetwork(input.InternalPort, input.ExternalPort, netType, input.ExternalIP, input.FQDN)
		if err != nil {
			uc.logger.Error(ctx, "failed to add network",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
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
		externalPort = network.ExternalPort()
		networkType = network.NetworkType().String()
		externalIP = network.ExternalIP()
		fqdn = network.FQDN()
		createdAt = network.CreatedAt().Format("2006-01-02T15:04:05Z")

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "add network completed",
		zap.Uint("container_id", input.ContainerID),
		zap.Uint("network_id", networkID),
		zap.String("network_type", networkType),
	)

	return &AddNetworkOutput{
		ContainerID:  input.ContainerID,
		NetworkID:    networkID,
		InternalPort: internalPort,
		ExternalPort: externalPort,
		NetworkType:  networkType,
		ExternalIP:   externalIP,
		FQDN:         fqdn,
		CreatedAt:    createdAt,
	}, nil
}
