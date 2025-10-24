package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

func TestDeleteProjectUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	txManager := db.NewStubTxManager()

	t.Run("성공: 프로젝트 삭제", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 1,
		}

		mockVolumeService.On("DeleteVolumesByProjectID", mock.Anything, input.ProjectID).Return(nil)
		mockProjectService.On("DeleteProject", mock.Anything, input.ProjectID).Return(nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, "Project deleted successfully", output.Message)

		mockVolumeService.AssertExpectations(t)
		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 볼륨 삭제 실패 시 프로젝트도 삭제되지 않음", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 1,
		}

		mockVolumeService.On("DeleteVolumesByProjectID", mock.Anything, input.ProjectID).Return(projecterrors.ErrDatabaseOperation)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrDatabaseOperation, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
		// mockProjectService는 호출되지 않아야 함
	})

	t.Run("성공: 볼륨이 없는 프로젝트 삭제", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 1,
		}

		mockVolumeService.On("DeleteVolumesByProjectID", mock.Anything, input.ProjectID).Return(nil)
		mockProjectService.On("DeleteProject", mock.Anything, input.ProjectID).Return(nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)

		mockVolumeService.AssertExpectations(t)
		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: ProjectID가 0", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 0,
		}

		mockVolumeService.On("DeleteVolumesByProjectID", mock.Anything, input.ProjectID).Return(nil)
		mockProjectService.On("DeleteProject", mock.Anything, input.ProjectID).Return(projecterrors.ErrInvalidProjectID)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidProjectID, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 프로젝트를 찾을 수 없음", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 999,
		}

		mockVolumeService.On("DeleteVolumesByProjectID", mock.Anything, input.ProjectID).Return(nil)
		mockProjectService.On("DeleteProject", mock.Anything, input.ProjectID).Return(projecterrors.ErrProjectNotFound)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrProjectNotFound, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 프로젝트 삭제 불가", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 1,
		}

		mockVolumeService.On("DeleteVolumesByProjectID", mock.Anything, input.ProjectID).Return(nil)
		mockProjectService.On("DeleteProject", mock.Anything, input.ProjectID).Return(projecterrors.ErrCannotDeleteProject)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrCannotDeleteProject, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 에러", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 1,
		}

		mockVolumeService.On("DeleteVolumesByProjectID", mock.Anything, input.ProjectID).Return(nil)
		mockProjectService.On("DeleteProject", mock.Anything, input.ProjectID).Return(errors.New("database error"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 이미 삭제된 프로젝트", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 1,
		}

		mockVolumeService.On("DeleteVolumesByProjectID", mock.Anything, input.ProjectID).Return(nil)
		mockProjectService.On("DeleteProject", mock.Anything, input.ProjectID).Return(projecterrors.ErrCannotModifyDeletedProject)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrCannotModifyDeletedProject, err)
		assert.Nil(t, output)

		mockVolumeService.AssertExpectations(t)
		mockProjectService.AssertExpectations(t)
	})
}
