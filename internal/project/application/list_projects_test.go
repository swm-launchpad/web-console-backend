package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

func TestListProjectsUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 사용자의 프로젝트 목록 조회", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewListProjectsUseCase(mockProjectService, testLogger)

		userID := uint(1)
		input := ListProjectsInput{
			UserID: userID,
		}

		slug1, _ := value.NewProjectSlug("p2025011812000044444444")
		slug2, _ := value.NewProjectSlug("p2025011812000055555555")
		projects := []*model.Project{
			createTestProjectWithVolumes(1, "첫 번째 프로젝트", *slug1, userID),
			createTestProjectWithVolumes(2, "두 번째 프로젝트", *slug2, userID),
		}

		mockProjectService.On("ListProjects", ctx, userID).Return(projects, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Len(t, output.Projects, 2)

		assert.Equal(t, uint(1), output.Projects[0].ProjectID)
		assert.Equal(t, "첫 번째 프로젝트", output.Projects[0].Name)
		assert.Equal(t, "p2025011812000044444444", output.Projects[0].Slug)
		assert.Equal(t, "active", output.Projects[0].Status)

		assert.Equal(t, uint(2), output.Projects[1].ProjectID)
		assert.Equal(t, "두 번째 프로젝트", output.Projects[1].Name)
		assert.Equal(t, "p2025011812000055555555", output.Projects[1].Slug)
		assert.Equal(t, "active", output.Projects[1].Status)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("성공: 빈 프로젝트 목록", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewListProjectsUseCase(mockProjectService, testLogger)

		userID := uint(1)
		input := ListProjectsInput{
			UserID: userID,
		}

		mockProjectService.On("ListProjects", ctx, userID).Return([]*model.Project{}, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Len(t, output.Projects, 0)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("성공: 단일 프로젝트", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewListProjectsUseCase(mockProjectService, testLogger)

		userID := uint(1)
		input := ListProjectsInput{
			UserID: userID,
		}

		slug, _ := value.NewProjectSlug("p2025011812000066666666")
		projects := []*model.Project{
			createTestProjectWithVolumes(1, "유일한 프로젝트", *slug, userID),
		}

		mockProjectService.On("ListProjects", ctx, userID).Return(projects, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Len(t, output.Projects, 1)
		assert.Equal(t, "유일한 프로젝트", output.Projects[0].Name)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("성공: Plan이 있는 프로젝트", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewListProjectsUseCase(mockProjectService, testLogger)

		userID := uint(1)
		input := ListProjectsInput{
			UserID: userID,
		}

		slug, _ := value.NewProjectSlug("p2025011812000077777777")
		project := createTestProjectWithVolumes(1, "운영 프로젝트", *slug, userID)

		plan := value.PlanPro
		_ = project.SetPlan(plan)

		projects := []*model.Project{project}

		mockProjectService.On("ListProjects", ctx, userID).Return(projects, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Len(t, output.Projects, 1)
		assert.Equal(t, "pro", output.Projects[0].Plan)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 에러", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewListProjectsUseCase(mockProjectService, testLogger)

		userID := uint(1)
		input := ListProjectsInput{
			UserID: userID,
		}

		mockProjectService.On("ListProjects", ctx, userID).Return([]*model.Project(nil), errors.New("database connection error"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database connection error")
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 프로젝트 서비스 에러", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewListProjectsUseCase(mockProjectService, testLogger)

		userID := uint(1)
		input := ListProjectsInput{
			UserID: userID,
		}

		mockProjectService.On("ListProjects", ctx, userID).Return([]*model.Project(nil), projecterrors.ErrDatabaseOperation)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrDatabaseOperation, err)
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
	})
}
