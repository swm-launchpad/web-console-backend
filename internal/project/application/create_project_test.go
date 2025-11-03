package application

import (
	"context"
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

func TestCreateProjectUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	txManager := db.NewStubTxManager()

	t.Run("성공: 유효한 입력으로 프로젝트 생성", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewCreateProjectUseCase(mockProjectService, txManager, testLogger)

		cpuLimit := uint32(1000)
		memoryLimit := uint32(2048)
		diskLimit := uint32(2048)
		trafficLimit := uint32(256)

		input := CreateProjectInput{
			Name:         "테스트 프로젝트",
			OwnerID:      1,
			CPULimit:     cpuLimit,
			MemoryLimit:  memoryLimit,
			DiskLimit:    diskLimit,
			TrafficLimit: trafficLimit,
		}

		slug, _ := value.NewProjectSlug("p2025011812000012345678")
		project := createTestProjectWithLimits(1, input.Name, *slug, input.OwnerID, &cpuLimit, &memoryLimit, &diskLimit, &trafficLimit)

		// Prepare expected limits
		expectedLimits, _ := value.NewResourceLimits(cpuLimit, memoryLimit, diskLimit, trafficLimit)

		mockProjectService.On("CreateProject", mock.Anything, input.Name, input.OwnerID, *expectedLimits, (*value.Plan)(nil)).Return(project, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(1), output.ProjectID)
		assert.Equal(t, input.Name, output.Name)
		assert.Equal(t, slug.String(), output.Slug)
		assert.Equal(t, "active", output.Status)
		assert.Equal(t, cpuLimit, output.CPULimit)
		assert.Equal(t, memoryLimit, output.MemoryLimit)
		assert.Equal(t, diskLimit, output.DiskLimit)
		assert.Equal(t, trafficLimit, output.TrafficLimit)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("성공: 최소 리소스 제한으로 프로젝트 생성", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewCreateProjectUseCase(mockProjectService, txManager, testLogger)

		cpuLimit := uint32(value.MinCPULimit)       // 500 millicores
		memoryLimit := uint32(value.MinMemoryLimit) // 512 Mi
		diskLimit := uint32(value.MinDiskLimit)     // 1024 Mi
		trafficLimit := uint32(value.MinTrafficLimit)

		input := CreateProjectInput{
			Name:         "최소 리소스 프로젝트",
			OwnerID:      1,
			CPULimit:     cpuLimit,
			MemoryLimit:  memoryLimit,
			DiskLimit:    diskLimit,
			TrafficLimit: trafficLimit,
		}

		slug, _ := value.NewProjectSlug("p2025011812000022222222")
		project := createTestProjectWithLimits(2, input.Name, *slug, input.OwnerID, &cpuLimit, &memoryLimit, &diskLimit, &trafficLimit)

		// Expected limits with minimum values
		expectedLimits, _ := value.NewResourceLimits(cpuLimit, memoryLimit, diskLimit, trafficLimit)

		mockProjectService.On("CreateProject", mock.Anything, input.Name, input.OwnerID, *expectedLimits, (*value.Plan)(nil)).Return(project, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(2), output.ProjectID)
		assert.Equal(t, cpuLimit, output.CPULimit)
		assert.Equal(t, memoryLimit, output.MemoryLimit)
		assert.Equal(t, diskLimit, output.DiskLimit)
		assert.Equal(t, trafficLimit, output.TrafficLimit)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("성공: Plan이 있는 프로젝트 생성", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewCreateProjectUseCase(mockProjectService, txManager, testLogger)

		plan := value.PlanEco
		cpuLimit := uint32(1000)
		memoryLimit := uint32(2048)
		diskLimit := uint32(2048)
		trafficLimit := uint32(256)

		input := CreateProjectInput{
			Name:         "프로젝트 with Plan",
			OwnerID:      1,
			Plan:         &plan,
			CPULimit:     cpuLimit,
			MemoryLimit:  memoryLimit,
			DiskLimit:    diskLimit,
			TrafficLimit: trafficLimit,
		}

		slug, _ := value.NewProjectSlug("p2025011812000033333333")
		project := createTestProjectWithLimits(3, input.Name, *slug, input.OwnerID, &cpuLimit, &memoryLimit, &diskLimit, &trafficLimit)
		// Set Plan
		_ = project.SetPlan(plan)

		// Prepare expected limits
		expectedLimits, _ := value.NewResourceLimits(cpuLimit, memoryLimit, diskLimit, trafficLimit)

		mockProjectService.On("CreateProject", mock.Anything, input.Name, input.OwnerID, *expectedLimits, &plan).Return(project, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(3), output.ProjectID)
		assert.Equal(t, plan.String(), output.Plan)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 빈 프로젝트 이름", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewCreateProjectUseCase(mockProjectService, txManager, testLogger)

		input := CreateProjectInput{
			Name:    "",
			OwnerID: 1,
		}

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
		mockProjectService.AssertNotCalled(t, "CreateProject")
	})

	t.Run("실패: OwnerID가 0", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewCreateProjectUseCase(mockProjectService, txManager, testLogger)

		cpuLimit := uint32(value.MinCPULimit)
		memoryLimit := uint32(value.MinMemoryLimit)
		diskLimit := uint32(value.MinDiskLimit)
		trafficLimit := uint32(value.MinTrafficLimit)

		input := CreateProjectInput{
			Name:         "테스트 프로젝트",
			OwnerID:      0,
			CPULimit:     cpuLimit,
			MemoryLimit:  memoryLimit,
			DiskLimit:    diskLimit,
			TrafficLimit: trafficLimit,
		}

		expectedLimits, _ := value.NewResourceLimits(cpuLimit, memoryLimit, diskLimit, trafficLimit)
		mockProjectService.On("CreateProject", mock.Anything, input.Name, input.OwnerID, *expectedLimits, (*value.Plan)(nil)).Return((*model.Project)(nil), projecterrors.ErrOwnerIDRequired)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 프로젝트 생성 서비스 에러", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewCreateProjectUseCase(mockProjectService, txManager, testLogger)

		cpuLimit := uint32(value.MinCPULimit)
		memoryLimit := uint32(value.MinMemoryLimit)
		diskLimit := uint32(value.MinDiskLimit)
		trafficLimit := uint32(value.MinTrafficLimit)

		input := CreateProjectInput{
			Name:         "테스트 프로젝트",
			OwnerID:      1,
			CPULimit:     cpuLimit,
			MemoryLimit:  memoryLimit,
			DiskLimit:    diskLimit,
			TrafficLimit: trafficLimit,
		}

		expectedLimits, _ := value.NewResourceLimits(cpuLimit, memoryLimit, diskLimit, trafficLimit)
		mockProjectService.On("CreateProject", mock.Anything, input.Name, input.OwnerID, *expectedLimits, (*value.Plan)(nil)).Return((*model.Project)(nil), projecterrors.ErrProjectCreationFailed)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrProjectCreationFailed, err)
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 프로젝트 이름 이미 존재", func(t *testing.T) {
		mockProjectService := new(service.MockProjectService)
		testLogger := logger.NewForTest()
		uc := NewCreateProjectUseCase(mockProjectService, txManager, testLogger)

		cpuLimit := uint32(value.MinCPULimit)
		memoryLimit := uint32(value.MinMemoryLimit)
		diskLimit := uint32(value.MinDiskLimit)
		trafficLimit := uint32(value.MinTrafficLimit)

		input := CreateProjectInput{
			Name:         "중복 프로젝트",
			OwnerID:      1,
			CPULimit:     cpuLimit,
			MemoryLimit:  memoryLimit,
			DiskLimit:    diskLimit,
			TrafficLimit: trafficLimit,
		}

		expectedLimits, _ := value.NewResourceLimits(cpuLimit, memoryLimit, diskLimit, trafficLimit)
		mockProjectService.On("CreateProject", mock.Anything, input.Name, input.OwnerID, *expectedLimits, (*value.Plan)(nil)).Return((*model.Project)(nil), projecterrors.ErrProjectNameExists)

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrProjectNameExists, err)
		assert.Nil(t, output)

		mockProjectService.AssertExpectations(t)
	})
}

func createTestProjectWithLimits(id uint, name string, slug value.ProjectSlug, ownerID uint, cpuLimit, memoryLimit, diskLimit *uint32, trafficLimit *uint32) *model.Project {
	cpu := uint32(value.MinCPULimit) // Changed from 100 to minimum valid value (500)
	if cpuLimit != nil {
		cpu = *cpuLimit
	}
	memory := uint32(512)
	if memoryLimit != nil {
		memory = *memoryLimit
	}
	disk := uint32(value.MinDiskLimit) // Use minimum valid value (1024)
	if diskLimit != nil {
		disk = *diskLimit
	}
	traffic := uint32(1048576) // 1TB minimum
	if trafficLimit != nil {
		traffic = *trafficLimit
	}
	limits, err := value.NewResourceLimits(cpu, memory, disk, traffic)
	if err != nil {
		panic(err) // This is a test helper, panic on error
	}

	project, err := model.NewProject(name, slug, ownerID, *limits, nil)
	if err != nil {
		panic(err) // This is a test helper, panic on error
	}
	project.SetProjectID(id)

	return project
}
