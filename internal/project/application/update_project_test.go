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
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

func stringPtr(s string) *string {
	return &s
}

func uint32Ptr(u uint32) *uint32 {
	return &u
}

func TestUpdateProjectUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	txManager := db.NewStubTxManager()

	t.Run("성공: 프로젝트 이름 업데이트", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewUpdateProjectUseCase(mockProjectService, txManager, testLogger)

		input := UpdateProjectInput{
			ProjectID: 1,
			Name:      stringPtr("새로운 프로젝트 이름"),
		}

		slug, _ := value.NewProjectSlug("p2025011812000012345678")
		updatedProject := createTestProjectWithVolumes(1, "새로운 프로젝트 이름", *slug, 1)
		// Note: SetUpdatedAt doesn't exist in the model, updatedAt is managed internally

		mockProjectService.On("UpdateProject", mock.Anything, input.ProjectID, mock.AnythingOfType("func(*model.Project) error")).Return(updatedProject, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(1), output.ProjectID)
		assert.Equal(t, "새로운 프로젝트 이름", output.Name)
		assert.Equal(t, "p2025011812000012345678", output.Slug)
		assert.NotEmpty(t, output.UpdatedAt)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("성공: FQDN과 Plan 업데이트", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewUpdateProjectUseCase(mockProjectService, txManager, testLogger)

		input := UpdateProjectInput{
			ProjectID: 1,
			FQDN:      stringPtr("my-project.example.com"),
			Plan:      stringPtr("premium"),
		}

		slug, _ := value.NewProjectSlug("p2025011812000012345678")
		updatedProject := createTestProjectWithVolumes(1, "테스트 프로젝트", *slug, 1)
		_ = updatedProject.SetFQDN(*input.FQDN)
		_ = updatedProject.SetPlan(*input.Plan)
		// Note: SetUpdatedAt doesn't exist in the model, updatedAt is managed internally

		mockProjectService.On("UpdateProject", mock.Anything, input.ProjectID, mock.AnythingOfType("func(*model.Project) error")).Return(updatedProject, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, "my-project.example.com", output.FQDN)
		assert.Equal(t, "premium", output.Plan)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("성공: 상태 업데이트", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewUpdateProjectUseCase(mockProjectService, txManager, testLogger)

		input := UpdateProjectInput{
			ProjectID: 1,
			Status:    stringPtr("running"),
		}

		slug, _ := value.NewProjectSlug("p2025011812000012345678")
		updatedProject := createTestProjectWithVolumes(1, "테스트 프로젝트", *slug, 1)
		_ = updatedProject.SetStatus(value.ProjectStatusActive)
		// Note: SetUpdatedAt doesn't exist in the model, updatedAt is managed internally

		mockProjectService.On("UpdateProject", mock.Anything, input.ProjectID, mock.AnythingOfType("func(*model.Project) error")).Return(updatedProject, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, "active", output.Status)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("성공: 리소스 제한 업데이트", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewUpdateProjectUseCase(mockProjectService, txManager, testLogger)

		input := UpdateProjectInput{
			ProjectID:    1,
			CPULimit:     uint32Ptr(2000),
			MemoryLimit:  uint32Ptr(4096),
			DiskLimit:    uint32Ptr(4096),
			TrafficLimit: uint32Ptr(1000000),
		}

		slug, _ := value.NewProjectSlug("p2025011812000012345678")
		updatedProject := createTestProjectWithVolumes(1, "테스트 프로젝트", *slug, 1)

		// Update resource limits
		limits, _ := value.NewResourceLimits(
			*input.CPULimit,
			*input.MemoryLimit,
			*input.DiskLimit,
			*input.TrafficLimit,
		)
		_ = updatedProject.SetResourceLimits(*limits)
		// Note: SetUpdatedAt doesn't exist in the model, updatedAt is managed internally

		mockProjectService.On("UpdateProject", mock.Anything, input.ProjectID, mock.AnythingOfType("func(*model.Project) error")).Return(updatedProject, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("성공: 모든 필드 업데이트", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewUpdateProjectUseCase(mockProjectService, txManager, testLogger)

		input := UpdateProjectInput{
			ProjectID:    1,
			Name:         stringPtr("완전히 새로운 프로젝트"),
			FQDN:         stringPtr("new.example.com"),
			Plan:         stringPtr("enterprise"),
			Status:       stringPtr("running"),
			CPULimit:     uint32Ptr(4000),
			MemoryLimit:  uint32Ptr(8192),
			DiskLimit:    uint32Ptr(8192),
			TrafficLimit: uint32Ptr(2000000),
		}

		slug, _ := value.NewProjectSlug("p2025011812000012345678")
		updatedProject := createTestProjectWithVolumes(1, "완전히 새로운 프로젝트", *slug, 1)
		_ = updatedProject.SetFQDN(*input.FQDN)
		_ = updatedProject.SetPlan(*input.Plan)
		_ = updatedProject.SetStatus(value.ProjectStatusActive)

		limits, _ := value.NewResourceLimits(
			*input.CPULimit,
			*input.MemoryLimit,
			*input.DiskLimit,
			*input.TrafficLimit,
		)
		_ = updatedProject.SetResourceLimits(*limits)
		// Note: SetUpdatedAt doesn't exist in the model, updatedAt is managed internally

		mockProjectService.On("UpdateProject", mock.Anything, input.ProjectID, mock.AnythingOfType("func(*model.Project) error")).Return(updatedProject, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, "완전히 새로운 프로젝트", output.Name)
		assert.Equal(t, "new.example.com", output.FQDN)
		assert.Equal(t, "enterprise", output.Plan)
		assert.Equal(t, "active", output.Status)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: ProjectID가 0", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewUpdateProjectUseCase(mockProjectService, txManager, testLogger)

		input := UpdateProjectInput{
			ProjectID: 0,
			Name:      stringPtr("새 이름"),
		}

		mockProjectService.On("UpdateProject", mock.Anything, input.ProjectID, mock.AnythingOfType("func(*model.Project) error")).Return((*model.Project)(nil), projecterrors.ErrInvalidProjectID)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidProjectID, err)
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 프로젝트를 찾을 수 없음", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewUpdateProjectUseCase(mockProjectService, txManager, testLogger)

		input := UpdateProjectInput{
			ProjectID: 999,
			Name:      stringPtr("새 이름"),
		}

		mockProjectService.On("UpdateProject", mock.Anything, input.ProjectID, mock.AnythingOfType("func(*model.Project) error")).Return((*model.Project)(nil), projecterrors.ErrProjectNotFound)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrProjectNotFound, err)
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 삭제된 프로젝트 수정 불가", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewUpdateProjectUseCase(mockProjectService, txManager, testLogger)

		input := UpdateProjectInput{
			ProjectID: 1,
			Name:      stringPtr("새 이름"),
		}

		mockProjectService.On("UpdateProject", mock.Anything, input.ProjectID, mock.AnythingOfType("func(*model.Project) error")).Return((*model.Project)(nil), projecterrors.ErrCannotModifyDeletedProject)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrCannotModifyDeletedProject, err)
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 잘못된 상태 전환", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewUpdateProjectUseCase(mockProjectService, txManager, testLogger)

		input := UpdateProjectInput{
			ProjectID: 1,
			Status:    stringPtr("invalid-status"),
		}

		mockProjectService.On("UpdateProject", mock.Anything, input.ProjectID, mock.AnythingOfType("func(*model.Project) error")).Return((*model.Project)(nil), projecterrors.ErrInvalidStatusTransition)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidStatusTransition, err)
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 에러", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewUpdateProjectUseCase(mockProjectService, txManager, testLogger)

		input := UpdateProjectInput{
			ProjectID: 1,
			Name:      stringPtr("새 이름"),
		}

		mockProjectService.On("UpdateProject", mock.Anything, input.ProjectID, mock.AnythingOfType("func(*model.Project) error")).Return((*model.Project)(nil), errors.New("database error"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
	})
}
