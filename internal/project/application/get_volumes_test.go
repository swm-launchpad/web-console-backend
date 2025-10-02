package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

func TestGetVolumesUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 특정 프로젝트의 볼륨 목록 조회", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewGetVolumesUseCase(mockVolumeService)

		projectID := uint(1)
		input := GetVolumesInput{
			ProjectID: projectID,
		}

		volumes := []*model.Volume{
			createTestVolume(1, projectID, "volume-1", 1024),
			createTestVolume(2, projectID, "volume-2", 2048),
		}

		mockVolumeService.On("ListVolumesByProjectID", ctx, projectID).Return(volumes, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Len(t, output.Volumes, 2)

		assert.Equal(t, uint(1), output.Volumes[0].VolumeID)
		assert.Equal(t, projectID, output.Volumes[0].ProjectID)
		assert.Equal(t, "volume-1", output.Volumes[0].Name)
		assert.Equal(t, uint32(1024), output.Volumes[0].Capacity)
		assert.NotEmpty(t, output.Volumes[0].CreatedAt)

		assert.Equal(t, uint(2), output.Volumes[1].VolumeID)
		assert.Equal(t, "volume-2", output.Volumes[1].Name)
		assert.Equal(t, uint32(2048), output.Volumes[1].Capacity)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("성공: 빈 볼륨 목록", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewGetVolumesUseCase(mockVolumeService)

		projectID := uint(1)
		input := GetVolumesInput{
			ProjectID: projectID,
		}

		mockVolumeService.On("ListVolumesByProjectID", ctx, projectID).Return([]*model.Volume{}, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Len(t, output.Volumes, 0)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("성공: 단일 볼륨", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewGetVolumesUseCase(mockVolumeService)

		projectID := uint(1)
		input := GetVolumesInput{
			ProjectID: projectID,
		}

		volumes := []*model.Volume{
			createTestVolume(1, projectID, "single-volume", 1024),
		}

		mockVolumeService.On("ListVolumesByProjectID", ctx, projectID).Return(volumes, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Len(t, output.Volumes, 1)
		assert.Equal(t, "single-volume", output.Volumes[0].Name)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 특정 프로젝트 볼륨 조회 에러", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewGetVolumesUseCase(mockVolumeService)

		projectID := uint(999)
		input := GetVolumesInput{
			ProjectID: projectID,
		}

		mockVolumeService.On("ListVolumesByProjectID", ctx, projectID).Return([]*model.Volume(nil), projecterrors.ErrProjectNotFound)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrProjectNotFound, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 프로젝트 ID가 0", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewGetVolumesUseCase(mockVolumeService)

		projectID := uint(0)
		input := GetVolumesInput{
			ProjectID: projectID,
		}

		mockVolumeService.On("ListVolumesByProjectID", ctx, projectID).Return([]*model.Volume(nil), projecterrors.ErrInvalidProjectID)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidProjectID, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 연결 오류", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewGetVolumesUseCase(mockVolumeService)

		projectID := uint(1)
		input := GetVolumesInput{
			ProjectID: projectID,
		}

		mockVolumeService.On("ListVolumesByProjectID", ctx, projectID).Return([]*model.Volume(nil), projecterrors.ErrDatabaseUnavailable)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrDatabaseUnavailable, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
	})
}
