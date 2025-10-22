package service

import (
	"context"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
	volumevalue "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume/value"
)

// VolumeService defines the interface for volume-related business logic
type VolumeService interface {
	// CreateVolume creates a new volume
	CreateVolume(ctx context.Context, projectID uint, name string, capacity uint32) (*model.Volume, error)

	// GetVolume retrieves a volume by ID
	GetVolume(ctx context.Context, volumeID uint) (*model.Volume, error)

	// GetVolumeBySlug retrieves a volume by slug
	GetVolumeBySlug(ctx context.Context, slug string) (*model.Volume, error)

	// ListVolumesByProjectID retrieves all volumes for a project
	ListVolumesByProjectID(ctx context.Context, projectID uint) ([]*model.Volume, error)

	// DeleteVolume removes a volume
	DeleteVolume(ctx context.Context, volumeID uint) error

	// DeleteVolumesByProjectID removes all volumes for a project (physical delete)
	DeleteVolumesByProjectID(ctx context.Context, projectID uint) error
}

// volumeService is the concrete implementation of VolumeService
type volumeService struct {
	volumeRepo    repository.VolumeRepository
	projectRepo   repository.ProjectRepository
	volumeSlugSvc VolumeSlugService
}

// NewVolumeService creates a new instance of VolumeService
func NewVolumeService(
	volumeRepo repository.VolumeRepository,
	projectRepo repository.ProjectRepository,
	volumeSlugSvc VolumeSlugService,
) VolumeService {
	return &volumeService{
		volumeRepo:    volumeRepo,
		projectRepo:   projectRepo,
		volumeSlugSvc: volumeSlugSvc,
	}
}

// CreateVolume creates a new volume with validation
func (s *volumeService) CreateVolume(ctx context.Context, projectID uint, name string, capacity uint32) (*model.Volume, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectID
	}

	// Lock project row to prevent race conditions (SELECT FOR UPDATE)
	// This ensures disk limit check is atomic with volume creation
	project, err := s.projectRepo.FindByIDForUpdate(ctx, projectID)
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

	// 비즈니스 로직: 프로젝트 디스크 제한 내에서 볼륨 용량 검증
	// 프로젝트에 설정된 디스크 제한을 초과하지 않도록 볼륨 총 용량을 제한
	diskLimit := project.Limits().DiskLimit()

	// Get total capacity efficiently using aggregate query
	totalExistingCapacity, err := s.volumeRepo.GetTotalCapacityByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Check if new volume would exceed project disk limit
	if totalExistingCapacity+capacity > diskLimit {
		return nil, projecterrors.ErrVolumeDiskLimitExceeded
	}

	// Create the volume
	volume, err := model.NewVolume(projectID, name, capacity)
	if err != nil {
		return nil, err
	}

	// Generate unique slug for the volume
	slug, err := s.volumeSlugSvc.GenerateSlug(ctx)
	if err != nil {
		return nil, err
	}
	volume.SetSlug(slug)

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

// GetVolumeBySlug retrieves a volume by slug
func (s *volumeService) GetVolumeBySlug(ctx context.Context, slug string) (*model.Volume, error) {
	// Validate slug format before repository lookup
	// This returns ErrSlugInvalidFormat for malformed slugs (wrong length/prefix)
	// which maps to 400 Bad Request instead of 404 Not Found
	_, err := volumevalue.NewVolumeSlug(slug)
	if err != nil {
		return nil, err
	}

	volume, err := s.volumeRepo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	return volume, nil
}

// ListVolumesByProjectID retrieves all volumes for a project
func (s *volumeService) ListVolumesByProjectID(ctx context.Context, projectID uint) ([]*model.Volume, error) {
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

// DeleteVolumesByProjectID removes all volumes for a project (physical delete)
func (s *volumeService) DeleteVolumesByProjectID(ctx context.Context, projectID uint) error {
	if projectID == 0 {
		return projecterrors.ErrInvalidProjectID
	}

	// Delete all volumes for the project
	if err := s.volumeRepo.DeleteByProjectID(ctx, projectID); err != nil {
		return err
	}

	return nil
}
