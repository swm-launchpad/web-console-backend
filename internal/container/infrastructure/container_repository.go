package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure/sqlc"
	"go.uber.org/zap"
)

type containerRepository struct {
	db      sqlc.DBTX
	queries *sqlc.Queries
	logger  logger.Logger
}

func NewContainerRepository(db sqlc.DBTX, log logger.Logger) repository.ContainerRepository {
	return &containerRepository{
		db:      db,
		queries: sqlc.New(db),
		logger:  log,
	}
}

func (r *containerRepository) Create(ctx context.Context, container *model.Container) error {
	r.logger.Info(ctx, "container repository create started",
		zap.Uint("project_id", container.ProjectID()),
		zap.String("name", container.Name()),
		zap.String("slug", container.Slug().String()),
	)

	qtx := r.queriesWithContext(ctx)

	// Marshal template config to JSON
	templateConfigJSON, err := json.Marshal(container.TemplateConfig())
	if err != nil {
		r.logger.Error(ctx, "container repository create invalid template config",
			zap.String("name", container.Name()),
			zap.Error(err),
		)
		return containererrors.ErrInvalidTemplateConfig
	}

	// Create container
	params := sqlc.CreateContainerParams{
		ProjectID:              uint32(container.ProjectID()),
		TemplateID:             uintToNullInt32(container.TemplateID()),
		Name:                   container.Name(),
		Slug:                   container.Slug().String(),
		StableWindow:           uint32PtrToNullInt32(container.StableWindow()),
		TemplateConfig:         templateConfigJSON,
		GithubInstallationID:   int64PtrToNullInt64(container.GitHubInstallationID()),
		GitRepositoryUrl:       sql.NullString{String: container.GitConfig().RepositoryURL(), Valid: container.GitConfig().RepositoryURL() != ""},
		GitBranch:              sql.NullString{String: container.GitConfig().Branch(), Valid: container.GitConfig().Branch() != ""},
		GitDirectoryPath:       stringPtrToNullString(container.GitConfig().DirectoryPath()),
		GitCommitHash:          stringPtrToNullString(container.GitCommitHash()),
		LastBuiltGitCommitHash: stringPtrToNullString(container.LastBuiltGitCommitHash()),
		NeedsBuild:             container.NeedsBuild(),
		CpuLimit:               uint32PtrToNullInt32(container.ResourceLimits().CPULimit()),
		MemoryLimit:            uint32PtrToNullInt32(container.ResourceLimits().MemoryLimit()),
		MonthlyBuildTime:       uint32PtrToNullInt32(container.MonthlyBuildTime()),
		MonthlyBuildCount:      uint32PtrToNullInt32(container.MonthlyBuildCount()),
		MonthlyUptime:          stringPtrToNullString(container.MonthlyUptime()),
		CreatedAt:              container.CreatedAt(),
		UpdatedAt:              timeToNullTime(container.UpdatedAt()),
	}

	result, err := qtx.CreateContainer(ctx, params)
	if err != nil {
		if isDuplicateError(err) {
			return containererrors.ErrSlugAlreadyExists
		}
		return containererrors.ErrDatabaseOperation
	}

	// Get the auto-generated ID
	containerID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	container.SetContainerID(uint(containerID))

	// Save environment variables
	for _, envVar := range container.EnvVars() {
		envParams := sqlc.CreateEnvVarParams{
			ContainerID: uint32(container.ContainerID()),
			Key:         envVar.Key(),
			Value:       sql.NullString{String: envVar.Value(), Valid: true},
			CreatedAt:   envVar.CreatedAt(),
			UpdatedAt:   timeToNullTime(envVar.UpdatedAt()),
		}
		result, err := qtx.CreateEnvVar(ctx, envParams)
		if err != nil {
			return containererrors.ErrDatabaseOperation
		}
		// Set the generated ID back to the domain entity
		if envVar.EnvVarID() == 0 {
			id, err := result.LastInsertId()
			if err != nil {
				return containererrors.ErrDatabaseOperation
			}
			envVar.SetEnvVarID(uint(id))
		}
	}

	// Save networks
	for _, network := range container.Networks() {
		netParams := sqlc.CreateNetworkParams{
			ContainerID:  uint32(container.ContainerID()),
			InternalPort: uint16PtrToNullInt32(network.InternalPort()),
			ExternalPort: uint16PtrToNullInt32(network.ExternalPort()),
			ExternalIp:   stringPtrToNullString(network.ExternalIP()),
			Fqdn:         stringPtrToNullString(network.FQDN()),
			Type:         sqlc.NetworksType(network.NetworkType().String()),
			CreatedAt:    network.CreatedAt(),
			UpdatedAt:    timeToNullTime(network.UpdatedAt()),
		}
		if _, err := qtx.CreateNetwork(ctx, netParams); err != nil {
			// Check for FQDN duplicate key error
			if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
				if strings.Contains(mysqlErr.Message, "uk_networks_fqdn") {
					return containererrors.ErrDuplicateFQDN
				}
			}
			return containererrors.ErrDatabaseOperation
		}
	}

	// Save secrets
	for _, secret := range container.Secrets() {
		secretParams := sqlc.CreateSecretParams{
			ContainerID: uint32(container.ContainerID()),
			Key:         secret.Key(),
			Value:       sql.NullString{String: secret.Value(), Valid: true},
			CreatedAt:   secret.CreatedAt(),
			UpdatedAt:   timeToNullTime(secret.UpdatedAt()),
		}
		result, err := qtx.CreateSecret(ctx, secretParams)
		if err != nil {
			return containererrors.ErrDatabaseOperation
		}
		// Set the generated ID back to the domain entity
		if secret.SecretID() == 0 {
			id, err := result.LastInsertId()
			if err != nil {
				return containererrors.ErrDatabaseOperation
			}
			secret.SetSecretID(uint(id))
		}
	}

	// Save build variables
	for _, buildVar := range container.BuildVars() {
		buildVarParams := sqlc.CreateBuildVarParams{
			ContainerID: uint32(container.ContainerID()),
			Key:         buildVar.Key(),
			Value:       sql.NullString{String: buildVar.Value(), Valid: true},
			CreatedAt:   buildVar.CreatedAt(),
			UpdatedAt:   timeToNullTime(buildVar.UpdatedAt()),
		}
		result, err := qtx.CreateBuildVar(ctx, buildVarParams)
		if err != nil {
			return containererrors.ErrDatabaseOperation
		}
		// Set the generated ID back to the domain entity
		if buildVar.BuildVarID() == 0 {
			id, err := result.LastInsertId()
			if err != nil {
				return containererrors.ErrDatabaseOperation
			}
			buildVar.SetBuildVarID(uint(id))
		}
	}

	// Save mounts
	for _, mount := range container.Mounts() {
		mountParams := sqlc.CreateMountParams{
			ContainerID: uint32(container.ContainerID()),
			VolumeID:    uint32(mount.VolumeID()),
			MountPath:   mount.MountPath(),
			CreatedAt:   mount.CreatedAt(),
			UpdatedAt:   timeToNullTime(mount.UpdatedAt()),
		}
		if _, err := qtx.CreateMount(ctx, mountParams); err != nil {
			r.logger.Error(ctx, "container repository create mount failed",
				zap.Uint("container_id", container.ContainerID()),
				zap.Error(err),
			)
			return containererrors.ErrDatabaseOperation
		}
	}

	r.logger.Info(ctx, "container repository create completed",
		zap.Uint("container_id", container.ContainerID()),
		zap.String("name", container.Name()),
	)
	return nil
}

func (r *containerRepository) Save(ctx context.Context, container *model.Container) error {
	qtx := r.queriesWithContext(ctx)

	// Marshal template config to JSON
	templateConfigJSON, err := json.Marshal(container.TemplateConfig())
	if err != nil {
		return containererrors.ErrInvalidTemplateConfig
	}

	// Update container
	params := sqlc.UpdateContainerParams{
		Name:                   container.Name(),
		StableWindow:           uint32PtrToNullInt32(container.StableWindow()),
		TemplateID:             uintToNullInt32(container.TemplateID()),
		TemplateConfig:         templateConfigJSON,
		GithubInstallationID:   int64PtrToNullInt64(container.GitHubInstallationID()),
		GitRepositoryUrl:       sql.NullString{String: container.GitConfig().RepositoryURL(), Valid: container.GitConfig().RepositoryURL() != ""},
		GitBranch:              sql.NullString{String: container.GitConfig().Branch(), Valid: container.GitConfig().Branch() != ""},
		GitDirectoryPath:       stringPtrToNullString(container.GitConfig().DirectoryPath()),
		GitCommitHash:          stringPtrToNullString(container.GitCommitHash()),
		LastBuiltGitCommitHash: stringPtrToNullString(container.LastBuiltGitCommitHash()),
		NeedsBuild:             container.NeedsBuild(),
		CpuLimit:               uint32PtrToNullInt32(container.ResourceLimits().CPULimit()),
		MemoryLimit:            uint32PtrToNullInt32(container.ResourceLimits().MemoryLimit()),
		MonthlyBuildTime:       uint32PtrToNullInt32(container.MonthlyBuildTime()),
		MonthlyBuildCount:      uint32PtrToNullInt32(container.MonthlyBuildCount()),
		MonthlyUptime:          stringPtrToNullString(container.MonthlyUptime()),
		UpdatedAt:              timeToNullTime(container.UpdatedAt()),
		ContainerID:            uint32(container.ContainerID()),
	}

	if _, err := qtx.UpdateContainer(ctx, params); err != nil {
		return containererrors.ErrDatabaseOperation
	}

	// Sync environment variables
	// Simple approach: delete all existing and re-insert
	// (More sophisticated approach would be to track changes, but this is simpler and safer)
	if _, err := qtx.DeleteEnvVarsByContainerID(ctx, uint32(container.ContainerID())); err != nil {
		return containererrors.ErrDatabaseOperation
	}

	for _, envVar := range container.EnvVars() {
		envParams := sqlc.CreateEnvVarParams{
			ContainerID: uint32(container.ContainerID()),
			Key:         envVar.Key(),
			Value:       sql.NullString{String: envVar.Value(), Valid: true},
			CreatedAt:   envVar.CreatedAt(),
			UpdatedAt:   timeToNullTime(envVar.UpdatedAt()),
		}
		result, err := qtx.CreateEnvVar(ctx, envParams)
		if err != nil {
			return containererrors.ErrDatabaseOperation
		}
		// Set the generated ID back to the domain entity
		if envVar.EnvVarID() == 0 {
			id, err := result.LastInsertId()
			if err != nil {
				return containererrors.ErrDatabaseOperation
			}
			envVar.SetEnvVarID(uint(id))
		}
	}

	// Sync networks (maintain soft-delete for FQDN ownership tracking)
	// Get existing networks from database
	existingNets, err := qtx.GetNetworksByContainerID(ctx, uint32(container.ContainerID()))
	if err != nil {
		return containererrors.ErrDatabaseOperation
	}

	// Create map of existing networks by network_id
	existingNetMap := make(map[uint]sqlc.Network)
	for _, net := range existingNets {
		existingNetMap[uint(net.NetworkID)] = net
	}

	// Create map of domain networks by network_id
	domainNetMap := make(map[uint]*model.Network)
	networks := container.Networks()
	for i := range networks {
		if networks[i].NetworkID() != 0 {
			domainNetMap[networks[i].NetworkID()] = &networks[i]
		}
	}

	// Process domain networks: INSERT new or UPDATE existing
	// Use pointer to modify the original network objects in the slice
	for i := range networks {
		network := &networks[i]
		if network.NetworkID() == 0 {
			// New network - INSERT
			netParams := sqlc.CreateNetworkParams{
				ContainerID:  uint32(container.ContainerID()),
				InternalPort: uint16PtrToNullInt32(network.InternalPort()),
				ExternalPort: uint16PtrToNullInt32(network.ExternalPort()),
				ExternalIp:   stringPtrToNullString(network.ExternalIP()),
				Fqdn:         stringPtrToNullString(network.FQDN()),
				Type:         sqlc.NetworksType(network.NetworkType().String()),
				CreatedAt:    network.CreatedAt(),
				UpdatedAt:    timeToNullTime(network.UpdatedAt()),
			}
			result, err := qtx.CreateNetwork(ctx, netParams)
			if err != nil {
				// Check for FQDN duplicate key error
				if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
					if strings.Contains(mysqlErr.Message, "uk_networks_fqdn") {
						return containererrors.ErrDuplicateFQDN
					}
				}
				return containererrors.ErrDatabaseOperation
			}
			// Set the generated network_id on the original object
			id, _ := result.LastInsertId()
			network.SetNetworkID(uint(id))
		} else {
			// Existing network - UPDATE
			updateParams := sqlc.UpdateNetworkParams{
				InternalPort: uint16PtrToNullInt32(network.InternalPort()),
				Type:         sqlc.NetworksType(network.NetworkType().String()),
				ExternalPort: uint16PtrToNullInt32(network.ExternalPort()),
				ExternalIp:   stringPtrToNullString(network.ExternalIP()),
				Fqdn:         stringPtrToNullString(network.FQDN()),
				UpdatedAt:    timeToNullTime(network.UpdatedAt()),
				NetworkID:    uint32(network.NetworkID()),
			}
			if _, err := qtx.UpdateNetwork(ctx, updateParams); err != nil {
				// Check for FQDN duplicate key error
				if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
					if strings.Contains(mysqlErr.Message, "uk_networks_fqdn") {
						return containererrors.ErrDuplicateFQDN
					}
				}
				return containererrors.ErrDatabaseOperation
			}
		}
	}

	// Soft-delete networks that exist in DB but not in domain
	for networkID := range existingNetMap {
		if _, exists := domainNetMap[networkID]; !exists {
			// This network was removed - soft delete it
			softDelParams := sqlc.SoftDeleteNetworkByIDParams{
				DeletedAt: sql.NullTime{Time: time.Now(), Valid: true},
				NetworkID: uint32(networkID),
			}
			if _, err := qtx.SoftDeleteNetworkByID(ctx, softDelParams); err != nil {
				return containererrors.ErrDatabaseOperation
			}
		}
	}

	// Sync secrets
	if _, err := qtx.DeleteSecretsByContainerID(ctx, uint32(container.ContainerID())); err != nil {
		return containererrors.ErrDatabaseOperation
	}

	for _, secret := range container.Secrets() {
		secretParams := sqlc.CreateSecretParams{
			ContainerID: uint32(container.ContainerID()),
			Key:         secret.Key(),
			Value:       sql.NullString{String: secret.Value(), Valid: true},
			CreatedAt:   secret.CreatedAt(),
			UpdatedAt:   timeToNullTime(secret.UpdatedAt()),
		}
		result, err := qtx.CreateSecret(ctx, secretParams)
		if err != nil {
			return containererrors.ErrDatabaseOperation
		}
		// Set the generated ID back to the domain entity
		if secret.SecretID() == 0 {
			id, err := result.LastInsertId()
			if err != nil {
				return containererrors.ErrDatabaseOperation
			}
			secret.SetSecretID(uint(id))
		}
	}

	// Sync build variables
	if _, err := qtx.DeleteBuildVarsByContainerID(ctx, uint32(container.ContainerID())); err != nil {
		return containererrors.ErrDatabaseOperation
	}

	for _, buildVar := range container.BuildVars() {
		buildVarParams := sqlc.CreateBuildVarParams{
			ContainerID: uint32(container.ContainerID()),
			Key:         buildVar.Key(),
			Value:       sql.NullString{String: buildVar.Value(), Valid: true},
			CreatedAt:   buildVar.CreatedAt(),
			UpdatedAt:   timeToNullTime(buildVar.UpdatedAt()),
		}
		result, err := qtx.CreateBuildVar(ctx, buildVarParams)
		if err != nil {
			return containererrors.ErrDatabaseOperation
		}
		// Set the generated ID back to the domain entity
		if buildVar.BuildVarID() == 0 {
			id, err := result.LastInsertId()
			if err != nil {
				return containererrors.ErrDatabaseOperation
			}
			buildVar.SetBuildVarID(uint(id))
		}
	}

	// Sync mounts
	if _, err := qtx.DeleteMountsByContainerID(ctx, uint32(container.ContainerID())); err != nil {
		return containererrors.ErrDatabaseOperation
	}

	for _, mount := range container.Mounts() {
		mountParams := sqlc.CreateMountParams{
			ContainerID: uint32(container.ContainerID()),
			VolumeID:    uint32(mount.VolumeID()),
			MountPath:   mount.MountPath(),
			CreatedAt:   mount.CreatedAt(),
			UpdatedAt:   timeToNullTime(mount.UpdatedAt()),
		}
		if _, err := qtx.CreateMount(ctx, mountParams); err != nil {
			return containererrors.ErrDatabaseOperation
		}
	}

	// Handle soft delete
	if container.IsDeleted() {
		deleteParams := sqlc.DeleteContainerParams{
			DeletedAt:   timePtrToNullTime(container.DeletedAt()),
			UpdatedAt:   timeToNullTime(container.UpdatedAt()),
			ContainerID: uint32(container.ContainerID()),
		}
		if _, err := qtx.DeleteContainer(ctx, deleteParams); err != nil {
			return containererrors.ErrDatabaseOperation
		}
	}

	return nil
}

func (r *containerRepository) FindByID(ctx context.Context, containerID uint) (*model.Container, error) {
	r.logger.Info(ctx, "container repository find by id started",
		zap.Uint("container_id", containerID),
	)

	qtx := r.queriesWithContext(ctx)

	// Get container
	sqlcContainer, err := qtx.GetContainerByID(ctx, uint32(containerID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Error(ctx, "container not found",
				zap.Uint("container_id", containerID),
				zap.Error(containererrors.ErrContainerNotFound),
			)
			return nil, containererrors.ErrContainerNotFound
		}
		r.logger.Error(ctx, "container repository find by id failed",
			zap.Uint("container_id", containerID),
			zap.Error(err),
		)
		return nil, containererrors.ErrDatabaseOperation
	}

	// Convert to domain model
	container, err := r.toDomainContainer(sqlcContainer)
	if err != nil {
		return nil, err
	}

	// Load environment variables
	sqlcEnvVars, err := qtx.GetEnvVarsByContainerID(ctx, uint32(containerID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, containererrors.ErrDatabaseOperation
	}

	if err := r.loadEnvVars(container, sqlcEnvVars); err != nil {
		return nil, err
	}

	// Load networks
	sqlcNetworks, err := qtx.GetNetworksByContainerID(ctx, uint32(containerID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, containererrors.ErrDatabaseOperation
	}

	if err := r.loadNetworks(container, sqlcNetworks); err != nil {
		return nil, err
	}

	// Load secrets
	sqlcSecrets, err := qtx.GetSecretsByContainerID(ctx, uint32(containerID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, containererrors.ErrDatabaseOperation
	}

	if err := r.loadSecrets(container, sqlcSecrets); err != nil {
		return nil, err
	}

	// Load build variables
	sqlcBuildVars, err := qtx.GetBuildVarsByContainerID(ctx, uint32(containerID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, containererrors.ErrDatabaseOperation
	}

	if err := r.loadBuildVars(container, sqlcBuildVars); err != nil {
		return nil, err
	}

	// Load mounts
	sqlcMounts, err := qtx.GetMountsByContainerID(ctx, uint32(containerID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, containererrors.ErrDatabaseOperation
	}

	if err := r.loadMounts(container, sqlcMounts); err != nil {
		return nil, err
	}

	r.logger.Info(ctx, "container repository find by id completed",
		zap.Uint("container_id", container.ContainerID()),
		zap.String("name", container.Name()),
	)
	return container, nil
}

func (r *containerRepository) FindByIDForUpdate(ctx context.Context, containerID uint) (*model.Container, error) {
	qtx := r.queriesWithContext(ctx)

	// Get container with row lock
	sqlcContainer, err := qtx.GetContainerByIDForUpdate(ctx, uint32(containerID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, containererrors.ErrContainerNotFound
		}
		return nil, containererrors.ErrDatabaseOperation
	}

	// Convert to domain model
	container, err := r.toDomainContainer(sqlcContainer)
	if err != nil {
		return nil, err
	}

	// Load environment variables
	sqlcEnvVars, err := qtx.GetEnvVarsByContainerID(ctx, uint32(containerID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, containererrors.ErrDatabaseOperation
	}

	if err := r.loadEnvVars(container, sqlcEnvVars); err != nil {
		return nil, err
	}

	// Load networks
	sqlcNetworks, err := qtx.GetNetworksByContainerID(ctx, uint32(containerID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, containererrors.ErrDatabaseOperation
	}

	if err := r.loadNetworks(container, sqlcNetworks); err != nil {
		return nil, err
	}

	// Load secrets
	sqlcSecrets, err := qtx.GetSecretsByContainerID(ctx, uint32(containerID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, containererrors.ErrDatabaseOperation
	}

	if err := r.loadSecrets(container, sqlcSecrets); err != nil {
		return nil, err
	}

	// Load build variables
	sqlcBuildVars, err := qtx.GetBuildVarsByContainerID(ctx, uint32(containerID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, containererrors.ErrDatabaseOperation
	}

	if err := r.loadBuildVars(container, sqlcBuildVars); err != nil {
		return nil, err
	}

	// Load mounts
	sqlcMounts, err := qtx.GetMountsByContainerID(ctx, uint32(containerID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, containererrors.ErrDatabaseOperation
	}

	if err := r.loadMounts(container, sqlcMounts); err != nil {
		return nil, err
	}

	return container, nil
}

func (r *containerRepository) FindByProjectID(ctx context.Context, projectID uint) ([]*model.Container, error) {
	qtx := r.queriesWithContext(ctx)

	sqlcContainers, err := qtx.ListContainersByProjectID(ctx, uint32(projectID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*model.Container{}, nil
		}
		r.logger.Error(ctx, "failed to list containers by project ID",
			zap.Error(err),
			zap.Uint("project_id", projectID),
		)
		return nil, containererrors.ErrDatabaseOperation
	}

	if len(sqlcContainers) == 0 {
		return []*model.Container{}, nil
	}

	// Convert to domain models
	containers := make([]*model.Container, 0, len(sqlcContainers))
	for _, sqlcContainer := range sqlcContainers {
		container, err := r.toDomainContainer(sqlcContainer)
		if err != nil {
			return nil, err
		}

		// Load environment variables for each container
		sqlcEnvVars, err := qtx.GetEnvVarsByContainerID(ctx, uint32(container.ContainerID()))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			r.logger.Error(ctx, "failed to get env vars by container ID",
				zap.Error(err),
				zap.Uint("container_id", container.ContainerID()),
			)
			return nil, containererrors.ErrDatabaseOperation
		}

		if err := r.loadEnvVars(container, sqlcEnvVars); err != nil {
			return nil, err
		}

		// Load networks for each container
		sqlcNetworks, err := qtx.GetNetworksByContainerID(ctx, uint32(container.ContainerID()))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			r.logger.Error(ctx, "failed to get networks by container ID",
				zap.Error(err),
				zap.Uint("container_id", container.ContainerID()),
			)
			return nil, containererrors.ErrDatabaseOperation
		}

		if err := r.loadNetworks(container, sqlcNetworks); err != nil {
			return nil, err
		}

		// Load secrets for each container
		sqlcSecrets, err := qtx.GetSecretsByContainerID(ctx, uint32(container.ContainerID()))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			r.logger.Error(ctx, "failed to get secrets by container ID",
				zap.Error(err),
				zap.Uint("container_id", container.ContainerID()),
			)
			return nil, containererrors.ErrDatabaseOperation
		}

		if err := r.loadSecrets(container, sqlcSecrets); err != nil {
			return nil, err
		}

		// Load build variables for each container
		sqlcBuildVars, err := qtx.GetBuildVarsByContainerID(ctx, uint32(container.ContainerID()))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			r.logger.Error(ctx, "failed to get build vars by container ID",
				zap.Error(err),
				zap.Uint("container_id", container.ContainerID()),
			)
			return nil, containererrors.ErrDatabaseOperation
		}

		if err := r.loadBuildVars(container, sqlcBuildVars); err != nil {
			return nil, err
		}

		// Load mounts for each container
		sqlcMounts, err := qtx.GetMountsByContainerID(ctx, uint32(container.ContainerID()))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			r.logger.Error(ctx, "failed to get mounts by container ID",
				zap.Error(err),
				zap.Uint("container_id", container.ContainerID()),
			)
			return nil, containererrors.ErrDatabaseOperation
		}

		if err := r.loadMounts(container, sqlcMounts); err != nil {
			return nil, err
		}

		containers = append(containers, container)
	}

	return containers, nil
}

func (r *containerRepository) FindBySlug(ctx context.Context, slug string) (*model.Container, error) {
	r.logger.Info(ctx, "container repository find by slug started",
		zap.String("slug", slug),
	)

	qtx := r.queriesWithContext(ctx)

	sqlcContainer, err := qtx.GetContainerBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Error(ctx, "container not found by slug",
				zap.String("slug", slug),
				zap.Error(containererrors.ErrContainerNotFound),
			)
			return nil, containererrors.ErrContainerNotFound
		}
		r.logger.Error(ctx, "container repository find by slug failed",
			zap.String("slug", slug),
			zap.Error(err),
		)
		return nil, containererrors.ErrDatabaseOperation
	}

	container, err := r.toDomainContainer(sqlcContainer)
	if err != nil {
		return nil, err
	}

	// Load environment variables
	sqlcEnvVars, err := qtx.GetEnvVarsByContainerID(ctx, uint32(container.ContainerID()))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, containererrors.ErrDatabaseOperation
	}

	if err := r.loadEnvVars(container, sqlcEnvVars); err != nil {
		return nil, err
	}

	// Load networks
	sqlcNetworks, err := qtx.GetNetworksByContainerID(ctx, uint32(container.ContainerID()))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, containererrors.ErrDatabaseOperation
	}

	if err := r.loadNetworks(container, sqlcNetworks); err != nil {
		return nil, err
	}

	// Load secrets
	sqlcSecrets, err := qtx.GetSecretsByContainerID(ctx, uint32(container.ContainerID()))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, containererrors.ErrDatabaseOperation
	}

	if err := r.loadSecrets(container, sqlcSecrets); err != nil {
		return nil, err
	}

	// Load build variables
	sqlcBuildVars, err := qtx.GetBuildVarsByContainerID(ctx, uint32(container.ContainerID()))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, containererrors.ErrDatabaseOperation
	}

	if err := r.loadBuildVars(container, sqlcBuildVars); err != nil {
		return nil, err
	}

	// Load mounts
	sqlcMounts, err := qtx.GetMountsByContainerID(ctx, uint32(container.ContainerID()))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, containererrors.ErrDatabaseOperation
	}

	if err := r.loadMounts(container, sqlcMounts); err != nil {
		return nil, err
	}

	r.logger.Info(ctx, "container repository find by slug completed",
		zap.Uint("container_id", container.ContainerID()),
		zap.String("slug", slug),
	)
	return container, nil
}

func (r *containerRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	result, err := r.queriesWithContext(ctx).ExistsBySlug(ctx, slug)
	if err != nil {
		return false, containererrors.ErrDatabaseOperation
	}
	return result, nil
}

func (r *containerRepository) ExistsByNameAndProjectID(ctx context.Context, projectID uint, name string) (bool, error) {
	params := sqlc.ExistsByNameAndProjectIDParams{
		ProjectID: uint32(projectID),
		Name:      name,
	}

	result, err := r.queriesWithContext(ctx).ExistsByNameAndProjectID(ctx, params)
	if err != nil {
		return false, containererrors.ErrDatabaseOperation
	}
	return result, nil
}

func (r *containerRepository) Delete(ctx context.Context, containerID uint) error {
	params := sqlc.DeleteContainerParams{
		DeletedAt:   sql.NullTime{Time: r.now(), Valid: true},
		UpdatedAt:   sql.NullTime{Time: r.now(), Valid: true},
		ContainerID: uint32(containerID),
	}

	_, err := r.queriesWithContext(ctx).DeleteContainer(ctx, params)
	if err != nil {
		return containererrors.ErrDatabaseOperation
	}

	return nil
}

func (r *containerRepository) DeleteByProjectID(ctx context.Context, projectID uint) error {
	now := r.now()
	params := sqlc.DeleteContainersByProjectIDParams{
		DeletedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt: sql.NullTime{Time: now, Valid: true},
		ProjectID: uint32(projectID),
	}

	_, err := r.queriesWithContext(ctx).DeleteContainersByProjectID(ctx, params)
	if err != nil {
		return containererrors.ErrDatabaseOperation
	}
	return nil
}

func (r *containerRepository) List(ctx context.Context, offset, limit int) ([]*model.Container, error) {
	params := sqlc.ListContainersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	sqlcContainers, err := r.queriesWithContext(ctx).ListContainers(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*model.Container{}, nil
		}
		return nil, containererrors.ErrDatabaseOperation
	}

	if len(sqlcContainers) == 0 {
		return []*model.Container{}, nil
	}

	containers := make([]*model.Container, 0, len(sqlcContainers))
	for _, sqlcContainer := range sqlcContainers {
		container, err := r.toDomainContainer(sqlcContainer)
		if err != nil {
			return nil, err
		}
		containers = append(containers, container)
	}

	return containers, nil
}

func (r *containerRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.queriesWithContext(ctx).CountContainers(ctx)
	if err != nil {
		return 0, containererrors.ErrDatabaseOperation
	}
	return count, nil
}

func (r *containerRepository) CountByProjectID(ctx context.Context, projectID uint) (int64, error) {
	count, err := r.queriesWithContext(ctx).CountContainersByProjectID(ctx, uint32(projectID))
	if err != nil {
		return 0, containererrors.ErrDatabaseOperation
	}
	return count, nil
}

func (r *containerRepository) CountByTemplateID(ctx context.Context, templateID uint) (int64, error) {
	count, err := r.queriesWithContext(ctx).CountContainersByTemplateID(ctx, sql.NullInt32{Int32: int32(templateID), Valid: true})
	if err != nil {
		return 0, containererrors.ErrDatabaseOperation
	}
	return count, nil
}

// Helper methods

func (r *containerRepository) loadEnvVars(container *model.Container, sqlcEnvVars []sqlc.EnvVar) error {
	for _, sqlcEnvVar := range sqlcEnvVars {
		envVar, err := r.toDomainEnvVar(sqlcEnvVar)
		if err != nil {
			return err
		}
		if err := container.AddEnvVarDirect(envVar); err != nil {
			r.logger.Error(context.Background(), "failed to add env var",
				zap.Error(err),
				zap.Uint("container_id", container.ContainerID()),
				zap.String("key", envVar.Key()),
			)
			return containererrors.ErrDatabaseOperation
		}
	}
	return nil
}

func (r *containerRepository) loadNetworks(container *model.Container, sqlcNetworks []sqlc.Network) error {
	for _, sqlcNetwork := range sqlcNetworks {
		network, err := r.toDomainNetwork(sqlcNetwork)
		if err != nil {
			return err
		}
		if err := container.AddNetworkDirect(network); err != nil {
			r.logger.Error(context.Background(), "failed to add network",
				zap.Error(err),
				zap.Uint("container_id", container.ContainerID()),
				zap.Uint32("network_id", sqlcNetwork.NetworkID),
			)
			return containererrors.ErrDatabaseOperation
		}
	}
	return nil
}

func (r *containerRepository) loadSecrets(container *model.Container, sqlcSecrets []sqlc.Secret) error {
	for _, sqlcSecret := range sqlcSecrets {
		secret, err := r.toDomainSecret(sqlcSecret)
		if err != nil {
			return err
		}
		if err := container.AddSecretDirect(secret); err != nil {
			r.logger.Error(context.Background(), "failed to add secret",
				zap.Error(err),
				zap.Uint("container_id", container.ContainerID()),
				zap.String("key", secret.Key()),
			)
			return containererrors.ErrDatabaseOperation
		}
	}
	return nil
}

func (r *containerRepository) loadBuildVars(container *model.Container, sqlcBuildVars []sqlc.GetBuildVarsByContainerIDRow) error {
	for _, sqlcBuildVar := range sqlcBuildVars {
		buildVar, err := r.toDomainBuildVarFromRow(sqlcBuildVar)
		if err != nil {
			return err
		}
		if err := container.AddBuildVarDirect(buildVar); err != nil {
			r.logger.Error(context.Background(), "failed to add build var",
				zap.Error(err),
				zap.Uint("container_id", container.ContainerID()),
				zap.String("key", buildVar.Key()),
			)
			return containererrors.ErrDatabaseOperation
		}
	}
	return nil
}

func (r *containerRepository) loadMounts(container *model.Container, sqlcMounts []sqlc.Mount) error {
	for _, sqlcMount := range sqlcMounts {
		mount, err := r.toDomainMount(sqlcMount)
		if err != nil {
			return err
		}
		if err := container.AddMountDirect(mount); err != nil {
			r.logger.Error(context.Background(), "failed to add mount",
				zap.Error(err),
				zap.Uint("container_id", container.ContainerID()),
				zap.Uint32("volume_id", sqlcMount.VolumeID),
			)
			return containererrors.ErrDatabaseOperation
		}
	}
	return nil
}

func (r *containerRepository) queriesWithContext(ctx context.Context) *sqlc.Queries {
	// Check if context has transaction
	if tx, ok := db.GetTx(ctx); ok && tx != nil {
		return r.queries.WithTx(tx)
	}

	return r.queries
}

func (r *containerRepository) toDomainContainer(sqlcContainer sqlc.Container) (*model.Container, error) {
	slug, err := value.NewContainerSlug(sqlcContainer.Slug)
	if err != nil {
		r.logger.Error(context.Background(), "failed to create container slug",
			zap.Error(err),
			zap.Uint32("container_id", sqlcContainer.ContainerID),
			zap.String("slug", sqlcContainer.Slug),
		)
		return nil, containererrors.ErrDatabaseOperation
	}

	// Parse template config
	var templateConfig map[string]interface{}
	if len(sqlcContainer.TemplateConfig) > 0 && string(sqlcContainer.TemplateConfig) != "null" {
		if err := json.Unmarshal(sqlcContainer.TemplateConfig, &templateConfig); err != nil {
			r.logger.Error(context.Background(), "failed to unmarshal template config",
				zap.Error(err),
				zap.Uint32("container_id", sqlcContainer.ContainerID),
				zap.String("template_config", string(sqlcContainer.TemplateConfig)),
			)
			return nil, containererrors.ErrDatabaseOperation
		}
	}

	// Parse git config
	gitURL := nullStringToString(sqlcContainer.GitRepositoryUrl)
	gitBranch := nullStringToString(sqlcContainer.GitBranch)
	if gitBranch == "" {
		gitBranch = "main"
	}
	var gitDirPath *string
	if sqlcContainer.GitDirectoryPath.Valid {
		gitDirPath = &sqlcContainer.GitDirectoryPath.String
	}

	gitConfig, err := value.NewGitConfig(gitURL, gitBranch, gitDirPath)
	if err != nil {
		r.logger.Error(context.Background(), "failed to create git config",
			zap.Error(err),
			zap.Uint32("container_id", sqlcContainer.ContainerID),
			zap.String("git_url", gitURL),
			zap.String("git_branch", gitBranch),
		)
		return nil, containererrors.ErrDatabaseOperation
	}

	// Parse resource limits
	resourceLimits, err := value.NewResourceLimits(
		nullInt32ToUint32Ptr(sqlcContainer.CpuLimit),
		nullInt32ToUint32Ptr(sqlcContainer.MemoryLimit),
	)
	if err != nil {
		r.logger.Error(context.Background(), "failed to create resource limits",
			zap.Error(err),
			zap.Uint32("container_id", sqlcContainer.ContainerID),
		)
		return nil, containererrors.ErrDatabaseOperation
	}

	// Parse monthly metrics
	var monthlyBuildTime, monthlyBuildCount *uint32
	if sqlcContainer.MonthlyBuildTime.Valid {
		val := uint32(sqlcContainer.MonthlyBuildTime.Int32)
		monthlyBuildTime = &val
	}
	if sqlcContainer.MonthlyBuildCount.Valid {
		val := uint32(sqlcContainer.MonthlyBuildCount.Int32)
		monthlyBuildCount = &val
	}
	var monthlyUptime *string
	if sqlcContainer.MonthlyUptime.Valid {
		monthlyUptime = &sqlcContainer.MonthlyUptime.String
	}

	var templateID *uint
	if sqlcContainer.TemplateID.Valid {
		val := uint(sqlcContainer.TemplateID.Int32)
		templateID = &val
	}

	var stableWindow *uint32
	if sqlcContainer.StableWindow.Valid {
		val := uint32(sqlcContainer.StableWindow.Int32)
		stableWindow = &val
	}

	var gitCommitHash *string
	if sqlcContainer.GitCommitHash.Valid {
		gitCommitHash = &sqlcContainer.GitCommitHash.String
	}

	var lastBuiltGitCommitHash *string
	if sqlcContainer.LastBuiltGitCommitHash.Valid {
		lastBuiltGitCommitHash = &sqlcContainer.LastBuiltGitCommitHash.String
	}

	githubInstallationID := nullInt64ToInt64Ptr(sqlcContainer.GithubInstallationID)

	container := model.ReconstructContainer(
		uint(sqlcContainer.ContainerID),
		uint(sqlcContainer.ProjectID),
		templateID,
		sqlcContainer.Name,
		slug,
		stableWindow,
		templateConfig,
		githubInstallationID,
		gitConfig,
		gitCommitHash,
		lastBuiltGitCommitHash,
		sqlcContainer.NeedsBuild,
		resourceLimits,
		monthlyBuildTime,
		monthlyBuildCount,
		monthlyUptime,
		sqlcContainer.IsDeleted,
		fromNullTime(sqlcContainer.DeletedAt),
		sqlcContainer.CreatedAt,
		fromNullTimeOrZero(sqlcContainer.UpdatedAt),
	)

	return container, nil
}

func (r *containerRepository) toDomainEnvVar(sqlcEnvVar sqlc.EnvVar) (*model.EnvVar, error) {
	envVar := model.ReconstructEnvVar(
		uint(sqlcEnvVar.EnvVarID),
		uint(sqlcEnvVar.ContainerID),
		sqlcEnvVar.Key,
		nullStringToString(sqlcEnvVar.Value),
		sqlcEnvVar.CreatedAt,
		fromNullTimeOrZero(sqlcEnvVar.UpdatedAt),
	)
	return envVar, nil
}

func (r *containerRepository) toDomainNetwork(sqlcNetwork sqlc.Network) (*model.Network, error) {
	var internalPort, externalPort *uint16
	if sqlcNetwork.InternalPort.Valid {
		val := uint16(sqlcNetwork.InternalPort.Int32)
		internalPort = &val
	}
	if sqlcNetwork.ExternalPort.Valid {
		val := uint16(sqlcNetwork.ExternalPort.Int32)
		externalPort = &val
	}

	var externalIP *string
	if sqlcNetwork.ExternalIp.Valid {
		externalIP = &sqlcNetwork.ExternalIp.String
	}

	var fqdn *string
	if sqlcNetwork.Fqdn.Valid {
		fqdn = &sqlcNetwork.Fqdn.String
	}

	networkType, err := value.NewNetworkType(string(sqlcNetwork.Type))
	if err != nil {
		return nil, containererrors.ErrDatabaseOperation
	}

	// Convert tektonEventID and expiresAt from sqlc types
	var tektonEventID *string
	if sqlcNetwork.TektonEventID.Valid {
		tektonEventID = &sqlcNetwork.TektonEventID.String
	}

	var expiresAt *time.Time
	if sqlcNetwork.ExpiresAt.Valid {
		expiresAt = &sqlcNetwork.ExpiresAt.Time
	}

	network := model.ReconstructNetwork(
		uint(sqlcNetwork.NetworkID),
		uint(sqlcNetwork.ContainerID),
		internalPort,
		externalPort,
		networkType,
		externalIP,
		fqdn,
		tektonEventID,
		expiresAt,
		sqlcNetwork.CreatedAt,
		fromNullTimeOrZero(sqlcNetwork.UpdatedAt),
	)

	return network, nil
}

func (r *containerRepository) toDomainSecret(sqlcSecret sqlc.Secret) (*model.Secret, error) {
	secret := model.ReconstructSecret(
		uint(sqlcSecret.SecretID),
		uint(sqlcSecret.ContainerID),
		sqlcSecret.Key,
		nullStringToString(sqlcSecret.Value),
		sqlcSecret.CreatedAt,
		fromNullTimeOrZero(sqlcSecret.UpdatedAt),
	)
	return secret, nil
}

func (r *containerRepository) toDomainBuildVarFromRow(sqlcBuildVar sqlc.GetBuildVarsByContainerIDRow) (*model.BuildVar, error) {
	buildVar := model.ReconstructBuildVar(
		uint(sqlcBuildVar.BuildVarID),
		uint(sqlcBuildVar.ContainerID),
		sqlcBuildVar.Key,
		nullStringToString(sqlcBuildVar.Value),
		sqlcBuildVar.CreatedAt,
		fromNullTimeOrZero(sqlcBuildVar.UpdatedAt),
	)
	return buildVar, nil
}

func (r *containerRepository) toDomainMount(sqlcMount sqlc.Mount) (*model.Mount, error) {
	mount := model.ReconstructMount(
		uint(sqlcMount.ContainerID),
		uint(sqlcMount.VolumeID),
		sqlcMount.MountPath,
		sqlcMount.CreatedAt,
		fromNullTimeOrZero(sqlcMount.UpdatedAt),
	)
	return mount, nil
}

func (r *containerRepository) now() time.Time {
	return time.Now()
}

// GetTotalResourceUsageByProject calculates the total resource usage for a project
func (r *containerRepository) GetTotalResourceUsageByProject(ctx context.Context, projectID uint) (totalCPU uint32, totalMemory uint32, err error) {
	query := `
		SELECT
			COALESCE(SUM(cpu_limit), 0) as total_cpu,
			COALESCE(SUM(memory_limit), 0) as total_memory
		FROM CONTAINERS
		WHERE project_id = ? AND is_deleted = FALSE
	`

	var totalCPUInt64, totalMemoryInt64 sql.NullInt64
	err = r.db.QueryRowContext(ctx, query, projectID).Scan(&totalCPUInt64, &totalMemoryInt64)
	if err != nil {
		return 0, 0, containererrors.ErrDatabaseOperation
	}

	return uint32(totalCPUInt64.Int64), uint32(totalMemoryInt64.Int64), nil
}

// CheckInternalPortExistsInProject checks if an internal port is already used by another container in the same project
func (r *containerRepository) CheckInternalPortExistsInProject(ctx context.Context, projectID uint, internalPort uint16) (bool, error) {
	r.logger.Debug(ctx, "checking internal port existence in project",
		zap.Uint("project_id", projectID),
		zap.Uint16("internal_port", internalPort),
	)

	qtx := r.queriesWithContext(ctx)
	result, err := qtx.CheckInternalPortExistsInProject(ctx, sqlc.CheckInternalPortExistsInProjectParams{
		ProjectID:    uint32(projectID),
		InternalPort: sql.NullInt32{Int32: int32(internalPort), Valid: true},
	})
	if err != nil {
		r.logger.Error(ctx, "failed to check internal port existence",
			zap.Uint("project_id", projectID),
			zap.Uint16("internal_port", internalPort),
			zap.Error(err),
		)
		return false, containererrors.ErrDatabaseOperation
	}

	return result, nil
}

// CheckFQDNExists checks if an FQDN is used anywhere in the system
func (r *containerRepository) CheckFQDNExists(ctx context.Context, fqdn string) (bool, error) {
	r.logger.Debug(ctx, "checking FQDN existence",
		zap.String("fqdn", fqdn),
	)

	qtx := r.queriesWithContext(ctx)
	result, err := qtx.CheckFQDNExists(ctx, sql.NullString{String: fqdn, Valid: true})
	if err != nil {
		r.logger.Error(ctx, "failed to check FQDN existence",
			zap.String("fqdn", fqdn),
			zap.Error(err),
		)
		return false, containererrors.ErrDatabaseOperation
	}

	return result, nil
}

// CheckFQDNExistsInOtherProject checks if FQDN is used by another project
func (r *containerRepository) CheckFQDNExistsInOtherProject(ctx context.Context, fqdn string, projectID uint) (bool, error) {
	r.logger.Debug(ctx, "checking FQDN existence in other project",
		zap.String("fqdn", fqdn),
		zap.Uint("exclude_project_id", projectID),
	)

	qtx := r.queriesWithContext(ctx)
	result, err := qtx.CheckFQDNExistsInOtherProject(ctx, sqlc.CheckFQDNExistsInOtherProjectParams{
		Fqdn:      sql.NullString{String: fqdn, Valid: true},
		ProjectID: uint32(projectID),
	})
	if err != nil {
		r.logger.Error(ctx, "failed to check FQDN existence in other project",
			zap.String("fqdn", fqdn),
			zap.Uint("exclude_project_id", projectID),
			zap.Error(err),
		)
		return false, containererrors.ErrDatabaseOperation
	}

	return result, nil
}

// CheckFQDNExistsInOtherProjectExcludingSelf checks if FQDN is used by another project, excluding self
func (r *containerRepository) CheckFQDNExistsInOtherProjectExcludingSelf(ctx context.Context, fqdn string, networkID uint, projectID uint) (bool, error) {
	r.logger.Debug(ctx, "checking FQDN existence in other project excluding self",
		zap.String("fqdn", fqdn),
		zap.Uint("network_id", networkID),
		zap.Uint("exclude_project_id", projectID),
	)

	qtx := r.queriesWithContext(ctx)
	result, err := qtx.CheckFQDNExistsInOtherProjectExcludingSelf(ctx, sqlc.CheckFQDNExistsInOtherProjectExcludingSelfParams{
		Fqdn:      sql.NullString{String: fqdn, Valid: true},
		NetworkID: uint32(networkID),
		ProjectID: uint32(projectID),
	})
	if err != nil {
		r.logger.Error(ctx, "failed to check FQDN existence in other project excluding self",
			zap.String("fqdn", fqdn),
			zap.Uint("network_id", networkID),
			zap.Uint("exclude_project_id", projectID),
			zap.Error(err),
		)
		return false, containererrors.ErrDatabaseOperation
	}

	return result, nil
}

// CheckFQDNExistsForProject checks FQDN with proper business rules
func (r *containerRepository) CheckFQDNExistsForProject(ctx context.Context, fqdn string, projectID uint) (bool, error) {
	r.logger.Debug(ctx, "checking FQDN existence for project with business rules",
		zap.String("fqdn", fqdn),
		zap.Uint("project_id", projectID),
	)

	qtx := r.queriesWithContext(ctx)
	result, err := qtx.CheckFQDNExistsForProject(ctx, sqlc.CheckFQDNExistsForProjectParams{
		Fqdn:        sql.NullString{String: fqdn, Valid: true},
		ProjectID:   uint32(projectID),
		ProjectID_2: uint32(projectID),
	})
	if err != nil {
		r.logger.Error(ctx, "failed to check FQDN existence for project",
			zap.String("fqdn", fqdn),
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return false, containererrors.ErrDatabaseOperation
	}

	return result, nil
}

// CheckFQDNExistsForProjectExcludingSelf checks FQDN with business rules, excluding self
func (r *containerRepository) CheckFQDNExistsForProjectExcludingSelf(ctx context.Context, fqdn string, networkID uint, projectID uint) (bool, error) {
	r.logger.Debug(ctx, "checking FQDN existence for project excluding self",
		zap.String("fqdn", fqdn),
		zap.Uint("network_id", networkID),
		zap.Uint("project_id", projectID),
	)

	qtx := r.queriesWithContext(ctx)
	result, err := qtx.CheckFQDNExistsForProjectExcludingSelf(ctx, sqlc.CheckFQDNExistsForProjectExcludingSelfParams{
		Fqdn:        sql.NullString{String: fqdn, Valid: true},
		NetworkID:   uint32(networkID),
		ProjectID:   uint32(projectID),
		ProjectID_2: uint32(projectID),
	})
	if err != nil {
		r.logger.Error(ctx, "failed to check FQDN existence for project excluding self",
			zap.String("fqdn", fqdn),
			zap.Uint("network_id", networkID),
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return false, containererrors.ErrDatabaseOperation
	}

	return result, nil
}

// CheckInternalPortExistsInProjectExcludingSelf checks if internal port exists in project, excluding self
func (r *containerRepository) CheckInternalPortExistsInProjectExcludingSelf(ctx context.Context, projectID uint, internalPort uint16, networkID uint) (bool, error) {
	r.logger.Debug(ctx, "checking internal port existence in project excluding self",
		zap.Uint("project_id", projectID),
		zap.Uint16("internal_port", internalPort),
		zap.Uint("network_id", networkID),
	)

	qtx := r.queriesWithContext(ctx)
	result, err := qtx.CheckInternalPortExistsInProjectExcludingSelf(ctx, sqlc.CheckInternalPortExistsInProjectExcludingSelfParams{
		ProjectID:    uint32(projectID),
		InternalPort: sql.NullInt32{Int32: int32(internalPort), Valid: true},
		NetworkID:    uint32(networkID),
	})
	if err != nil {
		r.logger.Error(ctx, "failed to check internal port existence in project excluding self",
			zap.Uint("project_id", projectID),
			zap.Uint16("internal_port", internalPort),
			zap.Uint("network_id", networkID),
			zap.Error(err),
		)
		return false, containererrors.ErrDatabaseOperation
	}

	return result, nil
}

// SoftDeleteNetworksByContainerID soft deletes all networks for a container
func (r *containerRepository) SoftDeleteNetworksByContainerID(ctx context.Context, containerID uint) error {
	r.logger.Debug(ctx, "soft deleting networks by container ID",
		zap.Uint("container_id", containerID),
	)

	qtx := r.queriesWithContext(ctx)
	_, err := qtx.SoftDeleteNetworksByContainerID(ctx, sqlc.SoftDeleteNetworksByContainerIDParams{
		DeletedAt:   sql.NullTime{Time: time.Now(), Valid: true},
		ContainerID: uint32(containerID),
	})
	if err != nil {
		r.logger.Error(ctx, "failed to soft delete networks by container ID",
			zap.Uint("container_id", containerID),
			zap.Error(err),
		)
		return containererrors.ErrDatabaseOperation
	}

	return nil
}

// FindAllSlugsByProjectIDIncludingDeleted retrieves all container slugs for a project including soft-deleted containers
func (r *containerRepository) FindAllSlugsByProjectIDIncludingDeleted(ctx context.Context, projectID uint) ([]string, error) {
	r.logger.Debug(ctx, "finding all container slugs by project ID including deleted",
		zap.Uint("project_id", projectID),
	)

	qtx := r.queriesWithContext(ctx)
	slugs, err := qtx.ListContainerSlugsByProjectIDIncludingDeleted(ctx, uint32(projectID))
	if err != nil {
		r.logger.Error(ctx, "failed to find container slugs by project ID",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, containererrors.ErrDatabaseOperation
	}

	return slugs, nil
}

// UpdateNetworkTektonEventID updates the Tekton PipelineRun name for a network
// and clears external_ip, external_port, expires_at (used when starting new NodePort creation)
func (r *containerRepository) UpdateNetworkTektonEventID(ctx context.Context, networkID uint, tektonEventID string) error {
	r.logger.Debug(ctx, "updating network tekton event ID",
		zap.Uint("network_id", networkID),
		zap.String("tekton_event_id", tektonEventID),
	)

	qtx := r.queriesWithContext(ctx)
	err := qtx.UpdateNetworkTektonEventID(ctx, sqlc.UpdateNetworkTektonEventIDParams{
		NetworkID:     uint32(networkID),
		TektonEventID: sql.NullString{String: tektonEventID, Valid: true},
	})
	if err != nil {
		r.logger.Error(ctx, "failed to update network tekton event ID",
			zap.Uint("network_id", networkID),
			zap.Error(err),
		)
		return containererrors.ErrDatabaseOperation
	}

	return nil
}

// UpdateNetworkNodePortResult updates the NodePort result fields (external_ip, external_port, expires_at)
// Used when PipelineRun completes and NodePort information becomes available
func (r *containerRepository) UpdateNetworkNodePortResult(ctx context.Context, networkID uint, externalIP string, externalPort uint16, expiresAt time.Time) error {
	r.logger.Debug(ctx, "updating network nodeport result",
		zap.Uint("network_id", networkID),
		zap.String("external_ip", externalIP),
		zap.Uint16("external_port", externalPort),
		zap.Time("expires_at", expiresAt),
	)

	qtx := r.queriesWithContext(ctx)
	err := qtx.UpdateNetworkNodePortResult(ctx, sqlc.UpdateNetworkNodePortResultParams{
		NetworkID:    uint32(networkID),
		ExternalIp:   sql.NullString{String: externalIP, Valid: true},
		ExternalPort: sql.NullInt32{Int32: int32(externalPort), Valid: true},
		ExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
	})
	if err != nil {
		r.logger.Error(ctx, "failed to update network nodeport result",
			zap.Uint("network_id", networkID),
			zap.Error(err),
		)
		return containererrors.ErrDatabaseOperation
	}

	return nil
}
