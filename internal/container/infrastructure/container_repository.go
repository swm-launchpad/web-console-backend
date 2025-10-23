package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure/sqlc"
)

type containerRepository struct {
	db      sqlc.DBTX
	queries *sqlc.Queries
}

func NewContainerRepository(db sqlc.DBTX) repository.ContainerRepository {
	return &containerRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *containerRepository) Create(ctx context.Context, container *model.Container) error {
	qtx := r.queriesWithContext(ctx)

	// Marshal template config to JSON
	templateConfigJSON, err := json.Marshal(container.TemplateConfig())
	if err != nil {
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
		if _, err := qtx.CreateEnvVar(ctx, envParams); err != nil {
			return containererrors.ErrDatabaseOperation
		}
	}

	// Save networks
	for _, network := range container.Networks() {
		netParams := sqlc.CreateNetworkParams{
			ContainerID:  uint32(container.ContainerID()),
			InternalPort: uint16PtrToNullInt16(network.InternalPort()),
			ExternalPort: uint16PtrToNullInt16(network.ExternalPort()),
			ExternalIp:   stringPtrToNullString(network.ExternalIP()),
			Fqdn:         stringPtrToNullString(network.FQDN()),
			Type:         sqlc.NetworksType(network.NetworkType().String()),
			CreatedAt:    network.CreatedAt(),
			UpdatedAt:    timeToNullTime(network.UpdatedAt()),
		}
		if _, err := qtx.CreateNetwork(ctx, netParams); err != nil {
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
		if _, err := qtx.CreateSecret(ctx, secretParams); err != nil {
			return containererrors.ErrDatabaseOperation
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
		if _, err := qtx.CreateBuildVar(ctx, buildVarParams); err != nil {
			return containererrors.ErrDatabaseOperation
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
			return containererrors.ErrDatabaseOperation
		}
	}

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
		if _, err := qtx.CreateEnvVar(ctx, envParams); err != nil {
			return containererrors.ErrDatabaseOperation
		}
	}

	// Sync networks
	if _, err := qtx.DeleteNetworksByContainerID(ctx, uint32(container.ContainerID())); err != nil {
		return containererrors.ErrDatabaseOperation
	}

	for _, network := range container.Networks() {
		netParams := sqlc.CreateNetworkParams{
			ContainerID:  uint32(container.ContainerID()),
			InternalPort: uint16PtrToNullInt16(network.InternalPort()),
			ExternalPort: uint16PtrToNullInt16(network.ExternalPort()),
			ExternalIp:   stringPtrToNullString(network.ExternalIP()),
			Fqdn:         stringPtrToNullString(network.FQDN()),
			Type:         sqlc.NetworksType(network.NetworkType().String()),
			CreatedAt:    network.CreatedAt(),
			UpdatedAt:    timeToNullTime(network.UpdatedAt()),
		}
		if _, err := qtx.CreateNetwork(ctx, netParams); err != nil {
			return containererrors.ErrDatabaseOperation
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
		if _, err := qtx.CreateSecret(ctx, secretParams); err != nil {
			return containererrors.ErrDatabaseOperation
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
		if _, err := qtx.CreateBuildVar(ctx, buildVarParams); err != nil {
			return containererrors.ErrDatabaseOperation
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
	qtx := r.queriesWithContext(ctx)

	// Get container
	sqlcContainer, err := qtx.GetContainerByID(ctx, uint32(containerID))
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
			return nil, containererrors.ErrDatabaseOperation
		}

		if err := r.loadEnvVars(container, sqlcEnvVars); err != nil {
			return nil, err
		}

		// Load networks for each container
		sqlcNetworks, err := qtx.GetNetworksByContainerID(ctx, uint32(container.ContainerID()))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, containererrors.ErrDatabaseOperation
		}

		if err := r.loadNetworks(container, sqlcNetworks); err != nil {
			return nil, err
		}

		// Load secrets for each container
		sqlcSecrets, err := qtx.GetSecretsByContainerID(ctx, uint32(container.ContainerID()))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, containererrors.ErrDatabaseOperation
		}

		if err := r.loadSecrets(container, sqlcSecrets); err != nil {
			return nil, err
		}

		// Load build variables for each container
		sqlcBuildVars, err := qtx.GetBuildVarsByContainerID(ctx, uint32(container.ContainerID()))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, containererrors.ErrDatabaseOperation
		}

		if err := r.loadBuildVars(container, sqlcBuildVars); err != nil {
			return nil, err
		}

		// Load mounts for each container
		sqlcMounts, err := qtx.GetMountsByContainerID(ctx, uint32(container.ContainerID()))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
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
	qtx := r.queriesWithContext(ctx)

	sqlcContainer, err := qtx.GetContainerBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, containererrors.ErrContainerNotFound
		}
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
			return containererrors.ErrDatabaseOperation
		}
	}
	return nil
}

func (r *containerRepository) loadSecrets(container *model.Container, sqlcSecrets []sqlc.GetSecretsByContainerIDRow) error {
	for _, sqlcSecret := range sqlcSecrets {
		secret, err := r.toDomainSecret(sqlcSecret)
		if err != nil {
			return err
		}
		if err := container.AddSecretDirect(secret); err != nil {
			return containererrors.ErrDatabaseOperation
		}
	}
	return nil
}

func (r *containerRepository) loadBuildVars(container *model.Container, sqlcBuildVars []sqlc.BuildVar) error {
	for _, sqlcBuildVar := range sqlcBuildVars {
		buildVar, err := r.toDomainBuildVarFromRow(sqlcBuildVar)
		if err != nil {
			return err
		}
		if err := container.AddBuildVarDirect(buildVar); err != nil {
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
		return nil, containererrors.ErrDatabaseOperation
	}

	// Parse template config
	var templateConfig map[string]interface{}
	if len(sqlcContainer.TemplateConfig) > 0 && string(sqlcContainer.TemplateConfig) != "null" {
		if err := json.Unmarshal(sqlcContainer.TemplateConfig, &templateConfig); err != nil {
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
		return nil, containererrors.ErrDatabaseOperation
	}

	// Parse resource limits
	resourceLimits, err := value.NewResourceLimits(
		nullInt32ToUint32Ptr(sqlcContainer.CpuLimit),
		nullInt32ToUint32Ptr(sqlcContainer.MemoryLimit),
	)
	if err != nil {
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
		val := uint16(sqlcNetwork.InternalPort.Int16)
		internalPort = &val
	}
	if sqlcNetwork.ExternalPort.Valid {
		val := uint16(sqlcNetwork.ExternalPort.Int16)
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

	network := model.ReconstructNetwork(
		uint(sqlcNetwork.NetworkID),
		uint(sqlcNetwork.ContainerID),
		internalPort,
		externalPort,
		networkType,
		externalIP,
		fqdn,
		sqlcNetwork.CreatedAt,
		fromNullTimeOrZero(sqlcNetwork.UpdatedAt),
	)

	return network, nil
}

func (r *containerRepository) toDomainSecret(sqlcSecret sqlc.GetSecretsByContainerIDRow) (*model.Secret, error) {
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

func (r *containerRepository) toDomainBuildVarFromRow(sqlcBuildVar sqlc.BuildVar) (*model.BuildVar, error) {
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
