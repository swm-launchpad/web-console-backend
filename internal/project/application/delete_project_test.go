package application

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

// createTestProjectForDeletion creates a test project with operation status "nothing"
func createTestProjectForDeletion(projectID uint, slug string) *model.Project {
	projectSlug, _ := value.NewProjectSlug(slug)
	status, _ := value.NewProjectStatus("active")
	limits, _ := value.NewResourceLimits(500, 512, 1024, 1024)
	freePlan := value.PlanFree
	return model.ReconstructProject(
		projectID,
		"Test Project",
		*projectSlug,
		status,
		value.ProjectOperationStatusNothing,
		nil,
		&freePlan,
		*limits,
		time.Now(),
		time.Now(),
		false,
		nil,
	)
}

func TestDeleteProjectUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	txManager := db.NewStubTxManager()

	t.Run("성공: 프로젝트 삭제", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		mockTektonCleanupClient := new(infrastructure.MockTektonCleanupClient)
		mockContainerSlugProvider := new(infrastructure.MockContainerSlugProvider)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, mockTektonCleanupClient, mockContainerSlugProvider, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 1,
		}

		testProject := createTestProjectForDeletion(input.ProjectID, "p20250105123456abcd1234")
		containerSlugs := []string{"c20251106123456abc", "c20251106789012def"}

		mockProjectService.On("GetProject", mock.Anything, input.ProjectID).Return(testProject, nil)
		mockVolumeService.On("DeleteVolumesByProjectID", mock.Anything, input.ProjectID).Return(nil)
		mockProjectService.On("DeleteProject", mock.Anything, input.ProjectID).Return(nil)
		mockContainerSlugProvider.On("GetContainerSlugsByProjectID", mock.Anything, input.ProjectID).Return(containerSlugs, nil)
		mockTektonCleanupClient.On("TriggerCleanup", mock.Anything, strconv.FormatUint(uint64(input.ProjectID), 10), "application", containerSlugs).Return(nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, "Project deleted successfully", output.Message)

		mockProjectService.AssertExpectations(t)
		mockVolumeService.AssertExpectations(t)
		mockContainerSlugProvider.AssertExpectations(t)
		mockTektonCleanupClient.AssertExpectations(t)
	})

	t.Run("실패: 볼륨 삭제 실패 시 프로젝트도 삭제되지 않음", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		mockTektonCleanupClient := new(infrastructure.MockTektonCleanupClient)
		mockContainerSlugProvider := new(infrastructure.MockContainerSlugProvider)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, mockTektonCleanupClient, mockContainerSlugProvider, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 1,
		}

		testProject := createTestProjectForDeletion(input.ProjectID, "p20250105123456abcd1234")
		mockProjectService.On("GetProject", mock.Anything, input.ProjectID).Return(testProject, nil)
		mockVolumeService.On("DeleteVolumesByProjectID", mock.Anything, input.ProjectID).Return(projecterrors.ErrDatabaseOperation)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrDatabaseOperation, err)
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
		mockVolumeService.AssertExpectations(t)
	})

	t.Run("성공: 볼륨이 없는 프로젝트 삭제", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		mockTektonCleanupClient := new(infrastructure.MockTektonCleanupClient)
		mockContainerSlugProvider := new(infrastructure.MockContainerSlugProvider)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, mockTektonCleanupClient, mockContainerSlugProvider, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 1,
		}

		testProject := createTestProjectForDeletion(input.ProjectID, "p20250105123456abcd1234")
		mockProjectService.On("GetProject", mock.Anything, input.ProjectID).Return(testProject, nil)
		mockVolumeService.On("DeleteVolumesByProjectID", mock.Anything, input.ProjectID).Return(nil)
		mockProjectService.On("DeleteProject", mock.Anything, input.ProjectID).Return(nil)
		mockContainerSlugProvider.On("GetContainerSlugsByProjectID", mock.Anything, input.ProjectID).Return([]string{}, nil)
		mockTektonCleanupClient.On("TriggerCleanup", mock.Anything, strconv.FormatUint(uint64(input.ProjectID), 10), "application", []string{}).Return(nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)

		mockProjectService.AssertExpectations(t)
		mockVolumeService.AssertExpectations(t)
		mockContainerSlugProvider.AssertExpectations(t)
		mockTektonCleanupClient.AssertExpectations(t)
	})

	t.Run("실패: ProjectID가 0", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		mockTektonCleanupClient := new(infrastructure.MockTektonCleanupClient)
		mockContainerSlugProvider := new(infrastructure.MockContainerSlugProvider)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, mockTektonCleanupClient, mockContainerSlugProvider, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 0,
		}

		testProject := createTestProjectForDeletion(input.ProjectID, "p20250105123456abcd1234")
		mockProjectService.On("GetProject", mock.Anything, input.ProjectID).Return(testProject, nil)
		mockVolumeService.On("DeleteVolumesByProjectID", mock.Anything, input.ProjectID).Return(nil)
		mockProjectService.On("DeleteProject", mock.Anything, input.ProjectID).Return(projecterrors.ErrInvalidProjectID)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidProjectID, err)
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 프로젝트를 찾을 수 없음", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		mockTektonCleanupClient := new(infrastructure.MockTektonCleanupClient)
		mockContainerSlugProvider := new(infrastructure.MockContainerSlugProvider)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, mockTektonCleanupClient, mockContainerSlugProvider, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 999,
		}

		mockProjectService.On("GetProject", mock.Anything, input.ProjectID).Return(nil, projecterrors.ErrProjectNotFound)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrProjectNotFound, err)
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 프로젝트 삭제 불가", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		mockTektonCleanupClient := new(infrastructure.MockTektonCleanupClient)
		mockContainerSlugProvider := new(infrastructure.MockContainerSlugProvider)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, mockTektonCleanupClient, mockContainerSlugProvider, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 1,
		}

		testProject := createTestProjectForDeletion(input.ProjectID, "p20250105123456abcd1234")
		mockProjectService.On("GetProject", mock.Anything, input.ProjectID).Return(testProject, nil)
		mockVolumeService.On("DeleteVolumesByProjectID", mock.Anything, input.ProjectID).Return(nil)
		mockProjectService.On("DeleteProject", mock.Anything, input.ProjectID).Return(projecterrors.ErrCannotDeleteProject)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrCannotDeleteProject, err)
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 에러", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		mockTektonCleanupClient := new(infrastructure.MockTektonCleanupClient)
		mockContainerSlugProvider := new(infrastructure.MockContainerSlugProvider)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, mockTektonCleanupClient, mockContainerSlugProvider, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 1,
		}

		testProject := createTestProjectForDeletion(input.ProjectID, "p20250105123456abcd1234")
		mockProjectService.On("GetProject", mock.Anything, input.ProjectID).Return(testProject, nil)
		mockVolumeService.On("DeleteVolumesByProjectID", mock.Anything, input.ProjectID).Return(nil)
		mockProjectService.On("DeleteProject", mock.Anything, input.ProjectID).Return(errors.New("database error"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
		mockVolumeService.AssertExpectations(t)
	})

	t.Run("실패: 이미 삭제된 프로젝트", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		mockVolumeService := new(service.MockVolumeService)
		mockTektonCleanupClient := new(infrastructure.MockTektonCleanupClient)
		mockContainerSlugProvider := new(infrastructure.MockContainerSlugProvider)
		testLogger := logger.NewForTest()
		uc := NewDeleteProjectUseCase(mockProjectService, mockVolumeService, mockTektonCleanupClient, mockContainerSlugProvider, txManager, testLogger)

		input := DeleteProjectInput{
			ProjectID: 1,
		}

		testProject := createTestProjectForDeletion(input.ProjectID, "p20250105123456abcd1234")
		mockProjectService.On("GetProject", mock.Anything, input.ProjectID).Return(testProject, nil)
		mockVolumeService.On("DeleteVolumesByProjectID", mock.Anything, input.ProjectID).Return(nil)
		mockProjectService.On("DeleteProject", mock.Anything, input.ProjectID).Return(projecterrors.ErrCannotModifyDeletedProject)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrCannotModifyDeletedProject, err)
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
		mockVolumeService.AssertExpectations(t)
	})
}
