package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

func TestAddVolumeUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	txManager := db.NewStubTxManager()

	t.Run("성공: 볼륨 추가", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewAddVolumeUseCase(mockVolumeService, txManager)

		input := AddVolumeInput{
			ProjectID: 1,
			Name:      "test-volume",
			Capacity:  1024,
		}

		volume := createTestVolume(1, input.ProjectID, input.Name, input.Capacity)

		mockVolumeService.On("CreateVolume", mock.Anything, input.ProjectID, input.Name, input.Capacity).Return(volume, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(1), output.VolumeID)
		assert.Equal(t, uint(1), output.ProjectID)
		assert.Equal(t, "test-volume", output.Name)
		assert.Equal(t, uint32(1024), output.Capacity)
		assert.NotEmpty(t, output.CreatedAt)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("성공: 큰 용량의 볼륨 추가", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewAddVolumeUseCase(mockVolumeService, txManager)

		input := AddVolumeInput{
			ProjectID: 1,
			Name:      "large-volume",
			Capacity:  2048, // 2GB (max allowed for MVP)
		}

		volume := createTestVolume(2, input.ProjectID, input.Name, input.Capacity)

		mockVolumeService.On("CreateVolume", mock.Anything, input.ProjectID, input.Name, input.Capacity).Return(volume, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint32(2048), output.Capacity)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("성공: 최소 용량의 볼륨 추가", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewAddVolumeUseCase(mockVolumeService, txManager)

		input := AddVolumeInput{
			ProjectID: 1,
			Name:      "small-volume",
			Capacity:  128, // 최소 용량
		}

		volume := createTestVolume(3, input.ProjectID, input.Name, input.Capacity)

		mockVolumeService.On("CreateVolume", mock.Anything, input.ProjectID, input.Name, input.Capacity).Return(volume, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint32(128), output.Capacity)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: ProjectID가 0", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewAddVolumeUseCase(mockVolumeService, txManager)

		input := AddVolumeInput{
			ProjectID: 0,
			Name:      "test-volume",
			Capacity:  1024,
		}

		mockVolumeService.On("CreateVolume", mock.Anything, input.ProjectID, input.Name, input.Capacity).Return(nil, projecterrors.ErrInvalidProjectID)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidProjectID, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 빈 볼륨 이름", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewAddVolumeUseCase(mockVolumeService, txManager)

		input := AddVolumeInput{
			ProjectID: 1,
			Name:      "",
			Capacity:  1024,
		}

		mockVolumeService.On("CreateVolume", mock.Anything, input.ProjectID, input.Name, input.Capacity).Return(nil, projecterrors.ErrVolumeNameRequired)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrVolumeNameRequired, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 용량이 너무 작음", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewAddVolumeUseCase(mockVolumeService, txManager)

		input := AddVolumeInput{
			ProjectID: 1,
			Name:      "small-volume",
			Capacity:  64, // 최소 용량 미만
		}

		mockVolumeService.On("CreateVolume", mock.Anything, input.ProjectID, input.Name, input.Capacity).Return(nil, projecterrors.ErrVolumeCapacityTooSmall)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrVolumeCapacityTooSmall, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 용량이 너무 큼", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewAddVolumeUseCase(mockVolumeService, txManager)

		input := AddVolumeInput{
			ProjectID: 1,
			Name:      "huge-volume",
			Capacity:  20480, // 최대 용량 초과
		}

		mockVolumeService.On("CreateVolume", mock.Anything, input.ProjectID, input.Name, input.Capacity).Return(nil, projecterrors.ErrVolumeCapacityExceeded)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrVolumeCapacityExceeded, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 프로젝트를 찾을 수 없음", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewAddVolumeUseCase(mockVolumeService, txManager)

		input := AddVolumeInput{
			ProjectID: 999,
			Name:      "test-volume",
			Capacity:  1024,
		}

		mockVolumeService.On("CreateVolume", mock.Anything, input.ProjectID, input.Name, input.Capacity).Return(nil, projecterrors.ErrProjectNotFound)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrProjectNotFound, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 중복된 볼륨 이름", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewAddVolumeUseCase(mockVolumeService, txManager)

		input := AddVolumeInput{
			ProjectID: 1,
			Name:      "existing-volume",
			Capacity:  1024,
		}

		mockVolumeService.On("CreateVolume", mock.Anything, input.ProjectID, input.Name, input.Capacity).Return(nil, projecterrors.ErrDuplicateVolumeName)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrDuplicateVolumeName, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 프로젝트 디스크 제한 초과", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewAddVolumeUseCase(mockVolumeService, txManager)

		input := AddVolumeInput{
			ProjectID: 1,
			Name:      "overflow-volume",
			Capacity:  1024,
		}

		mockVolumeService.On("CreateVolume", mock.Anything, input.ProjectID, input.Name, input.Capacity).Return(nil, projecterrors.ErrVolumeDiskLimitExceeded)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrVolumeDiskLimitExceeded, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 에러", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewAddVolumeUseCase(mockVolumeService, txManager)

		input := AddVolumeInput{
			ProjectID: 1,
			Name:      "test-volume",
			Capacity:  1024,
		}

		mockVolumeService.On("CreateVolume", mock.Anything, input.ProjectID, input.Name, input.Capacity).Return(nil, errors.New("database error"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
	})
}
