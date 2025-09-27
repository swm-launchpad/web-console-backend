package service

import (
	"context"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/repository"
)

// VolumeService defines the interface for volume-related business logic
type VolumeService interface {
	// CreateVolume creates a new volume
	CreateVolume(ctx context.Context, projectID uint, name string, capacity uint32) (*model.Volume, error)

	// GetVolume retrieves a volume by ID
	GetVolume(ctx context.Context, volumeID uint) (*model.Volume, error)

	// GetVolumesByProjectID retrieves all volumes for a project
	GetVolumesByProjectID(ctx context.Context, projectID uint) ([]*model.Volume, error)

	// UpdateVolume updates an existing volume
	UpdateVolume(ctx context.Context, volumeID uint, name string, capacity uint32) (*model.Volume, error)

	// DeleteVolume removes a volume
	DeleteVolume(ctx context.Context, volumeID uint) error

	// ListVolumes retrieves all volumes with pagination
	ListVolumes(ctx context.Context, offset, limit int) ([]*model.Volume, error)
}

// volumeService is the concrete implementation of VolumeService
type volumeService struct {
	volumeRepo  repository.VolumeRepository
	projectRepo repository.ProjectRepository
}

// NewVolumeService creates a new instance of VolumeService
func NewVolumeService(volumeRepo repository.VolumeRepository, projectRepo repository.ProjectRepository) VolumeService {
	return &volumeService{
		volumeRepo:  volumeRepo,
		projectRepo: projectRepo,
	}
}

// CreateVolume creates a new volume with validation
func (s *volumeService) CreateVolume(ctx context.Context, projectID uint, name string, capacity uint32) (*model.Volume, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectID
	}

	// Check if project exists
	_, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if err == projecterrors.ErrProjectNotFound {
			return nil, projecterrors.ErrInvalidProjectID
		}
		return nil, err
	}

	// Check for duplicate volume name in project
	exists, err := s.volumeRepo.ExistsByName(ctx, projectID, name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, projecterrors.ErrDuplicateVolumeName
	}

	// Create the volume
	volume, err := model.NewVolume(projectID, name, capacity)
	if err != nil {
		return nil, err
	}

	// Persist the volume
	if err := s.volumeRepo.Create(ctx, volume); err != nil {
		return nil, err
	}

	return volume, nil
}

// GetVolume retrieves a volume by ID
func (s *volumeService) GetVolume(ctx context.Context, volumeID uint) (*model.Volume, error) {
	if volumeID == 0 {
		return nil, projecterrors.ErrInvalidVolumeID
	}

	volume, err := s.volumeRepo.FindByID(ctx, volumeID)
	if err != nil {
		return nil, err
	}

	return volume, nil
}

// GetVolumesByProjectID retrieves all volumes for a project
func (s *volumeService) GetVolumesByProjectID(ctx context.Context, projectID uint) ([]*model.Volume, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectID
	}

	// Check if project exists
	_, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if err == projecterrors.ErrProjectNotFound {
			return nil, projecterrors.ErrInvalidProjectID
		}
		return nil, err
	}

	volumes, err := s.volumeRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return volumes, nil
}

// UpdateVolume updates an existing volume
func (s *volumeService) UpdateVolume(ctx context.Context, volumeID uint, name string, capacity uint32) (*model.Volume, error) {
	if volumeID == 0 {
		return nil, projecterrors.ErrInvalidVolumeID
	}

	// Retrieve the volume
	volume, err := s.volumeRepo.FindByID(ctx, volumeID)
	if err != nil {
		return nil, err
	}

	// Check for duplicate name (except for the volume being updated)
	if volume.GetName() != name {
		exists, err := s.volumeRepo.ExistsByName(ctx, volume.GetProjectID(), name)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, projecterrors.ErrDuplicateVolumeName
		}
	}

	// Update the volume
	if err := volume.Update(name, capacity); err != nil {
		return nil, err
	}

	// Save the updated volume
	if err := s.volumeRepo.Save(ctx, volume); err != nil {
		return nil, err
	}

	return volume, nil
}

// DeleteVolume removes a volume
func (s *volumeService) DeleteVolume(ctx context.Context, volumeID uint) error {
	if volumeID == 0 {
		return projecterrors.ErrInvalidVolumeID
	}

	// Check if volume exists
	_, err := s.volumeRepo.FindByID(ctx, volumeID)
	if err != nil {
		return err
	}

	// Delete the volume
	if err := s.volumeRepo.Delete(ctx, volumeID); err != nil {
		return err
	}

	return nil
}

// ListVolumes retrieves all volumes with pagination
func (s *volumeService) ListVolumes(ctx context.Context, offset, limit int) ([]*model.Volume, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if limit > 100 {
		limit = 100 // Maximum limit
	}

	volumes, err := s.volumeRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}

	return volumes, nil
}
