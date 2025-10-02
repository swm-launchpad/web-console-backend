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

func TestRemoveVolumeUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	txManager := db.NewStubTxManager()

	t.Run("성공: 볼륨 삭제", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewRemoveVolumeUseCase(mockVolumeService, txManager)

		input := RemoveVolumeInput{
			VolumeID: 1,
		}

		mockVolumeService.On("DeleteVolume", mock.Anything, input.VolumeID).Return(nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, "Volume removed successfully", output.Message)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("성공: 여러 볼륨 중 하나 삭제", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewRemoveVolumeUseCase(mockVolumeService, txManager)

		input := RemoveVolumeInput{
			VolumeID: 2,
		}

		mockVolumeService.On("DeleteVolume", mock.Anything, input.VolumeID).Return(nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: VolumeID가 0", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewRemoveVolumeUseCase(mockVolumeService, txManager)

		input := RemoveVolumeInput{
			VolumeID: 0,
		}

		mockVolumeService.On("DeleteVolume", mock.Anything, input.VolumeID).Return(projecterrors.ErrInvalidVolumeID)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidVolumeID, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 볼륨을 찾을 수 없음", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewRemoveVolumeUseCase(mockVolumeService, txManager)

		input := RemoveVolumeInput{
			VolumeID: 999,
		}

		mockVolumeService.On("DeleteVolume", mock.Anything, input.VolumeID).Return(projecterrors.ErrVolumeNotFound)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrVolumeNotFound, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 에러", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewRemoveVolumeUseCase(mockVolumeService, txManager)

		input := RemoveVolumeInput{
			VolumeID: 1,
		}

		mockVolumeService.On("DeleteVolume", mock.Anything, input.VolumeID).Return(errors.New("database error"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 동시성 충돌", func(t *testing.T) {
		mockVolumeService := new(service.MockVolumeService)
		uc := NewRemoveVolumeUseCase(mockVolumeService, txManager)

		input := RemoveVolumeInput{
			VolumeID: 1,
		}

		mockVolumeService.On("DeleteVolume", mock.Anything, input.VolumeID).Return(projecterrors.ErrConcurrentModification)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrConcurrentModification, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
	})
}
