package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
)

func defaultLimits() value.ResourceLimits {
	limits, _ := value.NewResourceLimits(100, 512, 1024, 1000)
	return *limits
}

func TestProjectService_CreateProject(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 프로젝트 생성", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		name := "테스트 프로젝트"
		slug, _ := value.NewProjectSlug("p2025011812000012345678")
		ownerID := uint(1)

		mockProjectRepo.On("ExistsByNameAndUserID", ctx, name, ownerID).Return(false, nil)
		mockSlugService.On("GenerateSlug", ctx).Return(*slug, nil)
		mockProjectRepo.On("Create", ctx, mock.MatchedBy(func(project *model.Project) bool {
			return project.Name() == name && project.Slug().String() == slug.String()
		})).Return(nil)

		project, err := service.CreateProject(ctx, name, ownerID, defaultLimits(), nil, nil)

		require.NoError(t, err)
		assert.NotNil(t, project)
		assert.Equal(t, name, project.Name())
		assert.Equal(t, slug.String(), project.Slug().String())

		mockProjectRepo.AssertExpectations(t)
		mockSlugService.AssertExpectations(t)
	})

	t.Run("실패: 빈 프로젝트 이름", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		project, err := service.CreateProject(ctx, "", 1, defaultLimits(), nil, nil)

		assert.Error(t, err)
		assert.Nil(t, project)

		mockProjectRepo.AssertNotCalled(t, "ExistsByNameAndUserID")
		mockSlugService.AssertNotCalled(t, "GenerateSlug")
		mockProjectRepo.AssertNotCalled(t, "Create")
	})

	t.Run("실패: OwnerID가 0", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		name := "테스트 프로젝트"

		project, err := service.CreateProject(ctx, name, 0, defaultLimits(), nil, nil)

		assert.Error(t, err)
		assert.Nil(t, project)

		mockProjectRepo.AssertNotCalled(t, "ExistsByNameAndUserID")
		mockSlugService.AssertNotCalled(t, "GenerateSlug")
		mockProjectRepo.AssertNotCalled(t, "Create")
	})

	t.Run("실패: 프로젝트 이름이 이미 존재", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		name := "기존 프로젝트"
		ownerID := uint(1)

		mockProjectRepo.On("ExistsByNameAndUserID", ctx, name, ownerID).Return(true, nil)

		project, err := service.CreateProject(ctx, name, ownerID, defaultLimits(), nil, nil)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrProjectNameExists, err)
		assert.Nil(t, project)

		mockProjectRepo.AssertExpectations(t)
		mockSlugService.AssertNotCalled(t, "GenerateSlug")
		mockProjectRepo.AssertNotCalled(t, "Create")
	})

	t.Run("실패: Slug 생성 실패", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		name := "테스트 프로젝트"
		ownerID := uint(1)

		mockProjectRepo.On("ExistsByNameAndUserID", ctx, name, ownerID).Return(false, nil)
		mockSlugService.On("GenerateSlug", ctx).Return(value.ProjectSlug{}, projecterrors.ErrSlugGenerationFailed)

		project, err := service.CreateProject(ctx, name, ownerID, defaultLimits(), nil, nil)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrSlugGenerationFailed, err)
		assert.Nil(t, project)

		mockProjectRepo.AssertExpectations(t)
		mockSlugService.AssertExpectations(t)
		mockProjectRepo.AssertNotCalled(t, "Create")
	})

	t.Run("실패: 저장소 생성 에러", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		name := "테스트 프로젝트"
		slug, _ := value.NewProjectSlug("p2025011812000012345678")
		ownerID := uint(1)

		mockProjectRepo.On("ExistsByNameAndUserID", ctx, name, ownerID).Return(false, nil)
		mockSlugService.On("GenerateSlug", ctx).Return(*slug, nil)
		mockProjectRepo.On("Create", ctx, mock.Anything).Return(errors.New("database error"))

		project, err := service.CreateProject(ctx, name, ownerID, defaultLimits(), nil, nil)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrProjectCreationFailed, err)
		assert.Nil(t, project)

		mockProjectRepo.AssertExpectations(t)
		mockSlugService.AssertExpectations(t)
	})
}

func TestProjectService_GetProject(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 ID로 프로젝트 조회", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		projectID := uint(1)
		slug, _ := value.NewProjectSlug("p2025011812000012345678")
		expectedProject := createTestProject(projectID, "테스트 프로젝트", *slug, 1)

		mockProjectRepo.On("FindByID", ctx, projectID).Return(expectedProject, nil)

		project, err := service.GetProject(ctx, projectID)

		require.NoError(t, err)
		assert.NotNil(t, project)
		assert.Equal(t, expectedProject, project)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: ProjectID가 0", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		project, err := service.GetProject(ctx, 0)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidProjectID, err)
		assert.Nil(t, project)

		mockProjectRepo.AssertNotCalled(t, "FindByID")
	})

	t.Run("실패: 프로젝트를 찾을 수 없음", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		projectID := uint(999)

		mockProjectRepo.On("FindByID", ctx, projectID).Return((*model.Project)(nil), projecterrors.ErrProjectNotFound)

		project, err := service.GetProject(ctx, projectID)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrProjectNotFound, err)
		assert.Nil(t, project)

		mockProjectRepo.AssertExpectations(t)
	})
}

func TestProjectService_UpdateProject(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 프로젝트 업데이트", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		projectID := uint(1)
		slug, _ := value.NewProjectSlug("p2025011812000012345678")
		project := createTestProject(projectID, "원래 이름", *slug, 1)

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)
		mockProjectRepo.On("Save", ctx, mock.Anything).Return(nil)

		updatedProject, err := service.UpdateProject(ctx, projectID, func(p *model.Project) error {
			return p.SetName("새로운 이름")
		})

		require.NoError(t, err)
		assert.NotNil(t, updatedProject)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: ProjectID가 0", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		updatedProject, err := service.UpdateProject(ctx, 0, func(p *model.Project) error {
			return nil
		})

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidProjectID, err)
		assert.Nil(t, updatedProject)

		mockProjectRepo.AssertNotCalled(t, "FindByID")
	})
}

func TestProjectService_DeleteProject(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 프로젝트 삭제", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		projectID := uint(1)
		slug, _ := value.NewProjectSlug("p2025011812000012345678")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, 1)

		mockProjectRepo.On("FindByID", ctx, projectID).Return(project, nil)
		mockProjectRepo.On("Save", ctx, mock.Anything).Return(nil)

		err := service.DeleteProject(ctx, projectID)

		require.NoError(t, err)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: ProjectID가 0", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		err := service.DeleteProject(ctx, 0)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidProjectID, err)

		mockProjectRepo.AssertNotCalled(t, "FindByID")
	})
}

func TestProjectService_ListProjects(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 사용자 프로젝트 목록 조회", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		userID := uint(1)
		slug1, _ := value.NewProjectSlug("p2025011812000099999991")
		slug2, _ := value.NewProjectSlug("p2025011812000099999992")
		expectedProjects := []*model.Project{
			createTestProject(1, "프로젝트 1", *slug1, userID),
			createTestProject(2, "프로젝트 2", *slug2, userID),
		}

		mockProjectRepo.On("FindByUserID", ctx, userID).Return(expectedProjects, nil)

		projects, err := service.ListProjects(ctx, userID)

		require.NoError(t, err)
		assert.Len(t, projects, 2)
		assert.Equal(t, expectedProjects, projects)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: UserID가 0", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		projects, err := service.ListProjects(ctx, 0)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidUserID, err)
		assert.Nil(t, projects)

		mockProjectRepo.AssertNotCalled(t, "FindByUserID")
	})
}

func TestProjectService_CountProjectsByUserID(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 활성 프로젝트 개수 조회", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		userID := uint(1)
		slug1, _ := value.NewProjectSlug("p2025011812000099999993")
		slug2, _ := value.NewProjectSlug("p2025011812000099999994")
		slug3, _ := value.NewProjectSlug("p2025011812000099999995")

		project1 := createTestProject(1, "프로젝트 1", *slug1, userID)
		project2 := createTestProject(2, "프로젝트 2", *slug2, userID)
		project3 := createTestProject(3, "프로젝트 3", *slug3, userID)
		_ = project3.SoftDelete() // 삭제된 프로젝트

		projects := []*model.Project{project1, project2, project3}

		mockProjectRepo.On("FindByUserID", ctx, userID).Return(projects, nil)

		count, err := service.CountProjectsByUserID(ctx, userID)

		require.NoError(t, err)
		assert.Equal(t, 2, count) // 삭제되지 않은 프로젝트만 카운트

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: UserID가 0", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		count, err := service.CountProjectsByUserID(ctx, 0)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidUserID, err)
		assert.Equal(t, 0, count)

		mockProjectRepo.AssertNotCalled(t, "FindByUserID")
	})
}

func TestProjectService_CheckProjectNameExists(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 프로젝트 이름 존재하지 않음", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		name := "새로운 프로젝트"
		userID := uint(1)

		mockProjectRepo.On("ExistsByNameAndUserID", ctx, name, userID).Return(false, nil)

		exists, err := service.CheckProjectNameExists(ctx, name, userID)

		require.NoError(t, err)
		assert.False(t, exists)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("성공: 프로젝트 이름 존재함", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		name := "기존 프로젝트"
		userID := uint(1)

		mockProjectRepo.On("ExistsByNameAndUserID", ctx, name, userID).Return(true, nil)

		exists, err := service.CheckProjectNameExists(ctx, name, userID)

		require.NoError(t, err)
		assert.True(t, exists)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: 빈 프로젝트 이름", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		exists, err := service.CheckProjectNameExists(ctx, "", 1)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrNameRequired, err)
		assert.False(t, exists)

		mockProjectRepo.AssertNotCalled(t, "ExistsByNameAndUserID")
	})

	t.Run("실패: UserID가 0", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		exists, err := service.CheckProjectNameExists(ctx, "테스트", 0)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidUserID, err)
		assert.False(t, exists)

		mockProjectRepo.AssertNotCalled(t, "ExistsByNameAndUserID")
	})
}

func TestProjectService_GetProjectBySlug(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 Slug로 프로젝트 조회", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		slug, _ := value.NewProjectSlug("p20250118120000abcd1234")
		expectedProject := createTestProject(1, "테스트 프로젝트", *slug, 1)

		mockProjectRepo.On("FindBySlug", ctx, slug.String()).Return(expectedProject, nil)

		project, err := service.GetProjectBySlug(ctx, slug.String())

		require.NoError(t, err)
		assert.NotNil(t, project)
		assert.Equal(t, expectedProject, project)

		mockProjectRepo.AssertExpectations(t)
	})

	t.Run("실패: Slug가 너무 짧음 (20자)", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		invalidSlug := "p20250118120000abcd" // 20 chars instead of 23

		project, err := service.GetProjectBySlug(ctx, invalidSlug)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrSlugInvalidLength, err)
		assert.Nil(t, project)

		mockProjectRepo.AssertNotCalled(t, "FindBySlug")
	})

	t.Run("실패: Slug가 너무 긺 (25자)", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		invalidSlug := "p20250118120000abcd123456" // 26 chars instead of 23

		project, err := service.GetProjectBySlug(ctx, invalidSlug)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrSlugInvalidLength, err)
		assert.Nil(t, project)

		mockProjectRepo.AssertNotCalled(t, "FindBySlug")
	})

	t.Run("실패: 잘못된 접두사 (c 대신 p)", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		invalidSlug := "c20250118120000abcd1234" // Container prefix instead of Project

		project, err := service.GetProjectBySlug(ctx, invalidSlug)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrSlugInvalidFormat, err)
		assert.Nil(t, project)

		mockProjectRepo.AssertNotCalled(t, "FindBySlug")
	})

	t.Run("실패: 잘못된 형식 (타임스탬프 부분에 문자)", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		invalidSlug := "p2025abc8120000abcd1234" // Letters in timestamp

		project, err := service.GetProjectBySlug(ctx, invalidSlug)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrSlugInvalidFormat, err)
		assert.Nil(t, project)

		mockProjectRepo.AssertNotCalled(t, "FindBySlug")
	})

	t.Run("실패: 프로젝트를 찾을 수 없음", func(t *testing.T) {
		mockProjectRepo := new(repository.MockProjectRepository)
		mockSlugService := new(MockSlugService)
		testLogger := logger.NewForTest()
		service := NewProjectService(mockProjectRepo, mockSlugService, testLogger)

		slug := "p20250118120000xyz91234"

		mockProjectRepo.On("FindBySlug", ctx, slug).Return((*model.Project)(nil), projecterrors.ErrProjectNotFound)

		project, err := service.GetProjectBySlug(ctx, slug)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrProjectNotFound, err)
		assert.Nil(t, project)

		mockProjectRepo.AssertExpectations(t)
	})
}

func createTestProject(id uint, name string, slug value.ProjectSlug, ownerID uint) *model.Project {
	project, _ := model.NewProject(name, slug, ownerID, defaultLimits(), nil, nil)
	project.SetProjectID(id)
	return project
}
