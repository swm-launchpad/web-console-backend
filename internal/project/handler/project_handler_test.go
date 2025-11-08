package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/settings"
	"github.com/swm-launchpad/web-console-backend/internal/project/application"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	volumemodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

func TestProjectHandler_CreateProject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	t.Run("성공: 프로젝트 생성 - MVP 정책 적용 확인", func(t *testing.T) {
		// Setup mocks
		mockProjectService := new(service.MockProjectService)
		mockPermissionService := new(service.MockPermissionService)
		txManager := db.NewStubTxManager()

		// Create real use cases with mocked dependencies
		mockVolumeService := new(service.MockVolumeService)
		testLogger := logger.NewForTest()
		createUseCase := application.NewCreateProjectUseCase(mockProjectService, txManager, testLogger)
		getUseCase := application.NewGetProjectUseCase(mockProjectService, mockVolumeService, testLogger)
		getBySlugUseCase := application.NewGetProjectBySlugUseCase(mockProjectService, mockVolumeService, testLogger)
		updateUseCase := application.NewUpdateProjectUseCase(mockProjectService, txManager, testLogger)
		mockTektonCleanupClient := new(infrastructure.MockTektonCleanupClient)
		mockContainerSlugProvider := new(infrastructure.MockContainerSlugProvider)
		deleteUseCase := application.NewDeleteProjectUseCase(mockProjectService, mockVolumeService, mockTektonCleanupClient, mockContainerSlugProvider, txManager, testLogger)
		listUseCase := application.NewListProjectsUseCase(mockProjectService, nil, testLogger)

		mockLogger, _ := logger.New(logger.Config{Level: "info", Format: "json"})
		// Settings service not needed for Pro plan test (only used for Free plan)
		var settingsService settings.SettingsService = nil
		handler := NewProjectHandler(
			createUseCase,
			getUseCase,
			getBySlugUseCase,
			updateUseCase,
			deleteUseCase,
			listUseCase,
			mockPermissionService,
			mockProjectService,
			settingsService,
			mockLogger,
		)

		userID := uint(1)
		projectName := "테스트 프로젝트"

		slug, _ := value.NewProjectSlug("p2025011812000012345678")
		project := createTestProject(uint(1), projectName, *slug, userID)

		// 사용자가 요청한 리소스 제한 (Beta tier limits: CPU 2 cores, Memory 4GB, Disk 3GB)
		cpuLimit := uint32(2000)
		memoryLimit := uint32(4096)
		diskLimit := uint32(3072) // Max in beta tier
		trafficLimit := uint32(524288)
		expectedLimits, _ := value.NewResourceLimits(cpuLimit, memoryLimit, diskLimit, trafficLimit)

		plan := value.PlanPro

		mockProjectService.On("CreateProject", ctx, projectName, userID, *expectedLimits, &plan).Return(project, nil)

		router := gin.New()
		router.POST("/projects", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.CreateProject(c)
		})

		// 사용자가 Plan, 리소스를 지정하면 그대로 사용됨
		reqBody := map[string]interface{}{
			"name":          projectName,
			"plan":          "pro",
			"cpu_limit":     2000,
			"memory_limit":  4096,
			"disk_limit":    3072, // Max in beta tier
			"traffic_limit": 524288,
		}
		jsonBody, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Logf("Response body: %s", w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["project_id"])
		assert.Equal(t, projectName, data["name"])
		assert.Equal(t, "p2025011812000012345678", data["slug"])

		mockProjectService.AssertExpectations(t)
	})

	t.Run("실패: 인증되지 않은 사용자", func(t *testing.T) {
		// Setup mocks
		mockProjectService := new(service.MockProjectService)
		mockPermissionService := new(service.MockPermissionService)
		txManager := db.NewStubTxManager()

		// Create real use cases with mocked dependencies
		mockVolumeService := new(service.MockVolumeService)
		testLogger := logger.NewForTest()
		createUseCase := application.NewCreateProjectUseCase(mockProjectService, txManager, testLogger)
		getUseCase := application.NewGetProjectUseCase(mockProjectService, mockVolumeService, testLogger)
		getBySlugUseCase := application.NewGetProjectBySlugUseCase(mockProjectService, mockVolumeService, testLogger)
		updateUseCase := application.NewUpdateProjectUseCase(mockProjectService, txManager, testLogger)
		mockTektonCleanupClient := new(infrastructure.MockTektonCleanupClient)
		mockContainerSlugProvider := new(infrastructure.MockContainerSlugProvider)
		deleteUseCase := application.NewDeleteProjectUseCase(mockProjectService, mockVolumeService, mockTektonCleanupClient, mockContainerSlugProvider, txManager, testLogger)
		listUseCase := application.NewListProjectsUseCase(mockProjectService, nil, testLogger)

		mockLogger, _ := logger.New(logger.Config{Level: "info", Format: "json"})
		// Settings service not needed for Pro plan test (only used for Free plan)
		var settingsService settings.SettingsService = nil
		handler := NewProjectHandler(
			createUseCase,
			getUseCase,
			getBySlugUseCase,
			updateUseCase,
			deleteUseCase,
			listUseCase,
			mockPermissionService,
			mockProjectService,
			settingsService,
			mockLogger,
		)

		router := gin.New()
		router.POST("/projects", handler.CreateProject)

		reqBody := map[string]interface{}{
			"name": "테스트 프로젝트",
		}
		jsonBody, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("실패: 프로젝트 한도 초과", func(t *testing.T) {
		// Setup mocks
		mockProjectService := new(service.MockProjectService)
		mockPermissionService := new(service.MockPermissionService)
		txManager := db.NewStubTxManager()

		// Create real use cases with mocked dependencies
		mockVolumeService := new(service.MockVolumeService)
		testLogger := logger.NewForTest()
		createUseCase := application.NewCreateProjectUseCase(mockProjectService, txManager, testLogger)
		getUseCase := application.NewGetProjectUseCase(mockProjectService, mockVolumeService, testLogger)
		getBySlugUseCase := application.NewGetProjectBySlugUseCase(mockProjectService, mockVolumeService, testLogger)
		updateUseCase := application.NewUpdateProjectUseCase(mockProjectService, txManager, testLogger)
		mockTektonCleanupClient := new(infrastructure.MockTektonCleanupClient)
		mockContainerSlugProvider := new(infrastructure.MockContainerSlugProvider)
		deleteUseCase := application.NewDeleteProjectUseCase(mockProjectService, mockVolumeService, mockTektonCleanupClient, mockContainerSlugProvider, txManager, testLogger)
		listUseCase := application.NewListProjectsUseCase(mockProjectService, nil, testLogger)

		mockLogger, _ := logger.New(logger.Config{Level: "info", Format: "json"})
		// Settings service not needed for Pro plan test (only used for Free plan)
		var settingsService settings.SettingsService = nil
		handler := NewProjectHandler(
			createUseCase,
			getUseCase,
			getBySlugUseCase,
			updateUseCase,
			deleteUseCase,
			listUseCase,
			mockPermissionService,
			mockProjectService,
			settingsService,
			mockLogger,
		)

		userID := uint(1)
		projectName := "테스트 프로젝트"

		// Mock CreateProject to return project limit exceeded error
		cpuLimit := uint32(1000)
		memoryLimit := uint32(2048)
		diskLimit := uint32(2048)
		trafficLimit := uint32(1048576)
		expectedLimits, _ := value.NewResourceLimits(cpuLimit, memoryLimit, diskLimit, trafficLimit)
		plan := value.PlanEco
		mockProjectService.On("CreateProject", ctx, projectName, userID, *expectedLimits, &plan).Return(nil, projecterrors.ErrProjectLimitExceeded)

		router := gin.New()
		router.POST("/projects", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.CreateProject(c)
		})

		reqBody := map[string]interface{}{
			"name":          projectName,
			"cpu_limit":     1000,
			"memory_limit":  2048,
			"disk_limit":    2048,
			"traffic_limit": 1048576,
		}
		jsonBody, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "PROJECT_LIMIT_EXCEEDED", errorData["code"])

		mockProjectService.AssertExpectations(t)
	})
}

func TestProjectHandler_GetProject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	t.Run("성공: ID로 프로젝트 조회", func(t *testing.T) {
		// Setup mocks
		mockProjectService := new(service.MockProjectService)
		mockPermissionService := new(service.MockPermissionService)
		txManager := db.NewStubTxManager()

		// Create real use cases with mocked dependencies
		mockVolumeService := new(service.MockVolumeService)
		testLogger := logger.NewForTest()
		createUseCase := application.NewCreateProjectUseCase(mockProjectService, txManager, testLogger)
		getUseCase := application.NewGetProjectUseCase(mockProjectService, mockVolumeService, testLogger)
		getBySlugUseCase := application.NewGetProjectBySlugUseCase(mockProjectService, mockVolumeService, testLogger)
		updateUseCase := application.NewUpdateProjectUseCase(mockProjectService, txManager, testLogger)
		mockTektonCleanupClient := new(infrastructure.MockTektonCleanupClient)
		mockContainerSlugProvider := new(infrastructure.MockContainerSlugProvider)
		deleteUseCase := application.NewDeleteProjectUseCase(mockProjectService, mockVolumeService, mockTektonCleanupClient, mockContainerSlugProvider, txManager, testLogger)
		listUseCase := application.NewListProjectsUseCase(mockProjectService, nil, testLogger)

		mockLogger, _ := logger.New(logger.Config{Level: "info", Format: "json"})
		// Settings service not needed for Pro plan test (only used for Free plan)
		var settingsService settings.SettingsService = nil
		handler := NewProjectHandler(
			createUseCase,
			getUseCase,
			getBySlugUseCase,
			updateUseCase,
			deleteUseCase,
			listUseCase,
			mockPermissionService,
			mockProjectService,
			settingsService,
			mockLogger,
		)

		userID := uint(1)
		projectID := uint(1)

		slug, _ := value.NewProjectSlug("p2025011812000087654321")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, userID)

		mockProjectService.On("GetProjectBySlug", ctx, slug.String()).Return(project, nil)
		mockVolumeService.On("ListVolumesByProjectID", ctx, projectID).Return([]*volumemodel.Volume{}, nil)
		mockPermissionService.On("CanUserAccessProject", ctx, userID, projectID).Return(nil)

		router := gin.New()
		router.GET("/projects/:slug", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetProject(c)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/projects/"+slug.String(), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(projectID), data["project_id"])
		assert.Equal(t, "테스트 프로젝트", data["name"])

		mockProjectService.AssertExpectations(t)
		mockPermissionService.AssertExpectations(t)
	})

	t.Run("실패: 권한 없음", func(t *testing.T) {
		// Setup mocks
		mockProjectService := new(service.MockProjectService)
		mockPermissionService := new(service.MockPermissionService)
		txManager := db.NewStubTxManager()

		// Create real use cases with mocked dependencies
		mockVolumeService := new(service.MockVolumeService)
		testLogger := logger.NewForTest()
		createUseCase := application.NewCreateProjectUseCase(mockProjectService, txManager, testLogger)
		getUseCase := application.NewGetProjectUseCase(mockProjectService, mockVolumeService, testLogger)
		getBySlugUseCase := application.NewGetProjectBySlugUseCase(mockProjectService, mockVolumeService, testLogger)
		updateUseCase := application.NewUpdateProjectUseCase(mockProjectService, txManager, testLogger)
		mockTektonCleanupClient := new(infrastructure.MockTektonCleanupClient)
		mockContainerSlugProvider := new(infrastructure.MockContainerSlugProvider)
		deleteUseCase := application.NewDeleteProjectUseCase(mockProjectService, mockVolumeService, mockTektonCleanupClient, mockContainerSlugProvider, txManager, testLogger)
		listUseCase := application.NewListProjectsUseCase(mockProjectService, nil, testLogger)

		mockLogger, _ := logger.New(logger.Config{Level: "info", Format: "json"})
		// Settings service not needed for Pro plan test (only used for Free plan)
		var settingsService settings.SettingsService = nil
		handler := NewProjectHandler(
			createUseCase,
			getUseCase,
			getBySlugUseCase,
			updateUseCase,
			deleteUseCase,
			listUseCase,
			mockPermissionService,
			mockProjectService,
			settingsService,
			mockLogger,
		)

		userID := uint(1)
		projectID := uint(1)

		slug, _ := value.NewProjectSlug("p2025011812000087654321")
		project := createTestProject(projectID, "테스트 프로젝트", *slug, uint(2)) // Different owner

		mockProjectService.On("GetProjectBySlug", ctx, slug.String()).Return(project, nil)
		mockVolumeService.On("ListVolumesByProjectID", ctx, projectID).Return([]*volumemodel.Volume{}, nil)
		mockPermissionService.On("CanUserAccessProject", ctx, userID, projectID).Return(projecterrors.ErrPermissionDenied)

		router := gin.New()
		router.GET("/projects/:slug", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetProject(c)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/projects/"+slug.String(), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code) // Should return not found for security

		mockProjectService.AssertExpectations(t)
		mockPermissionService.AssertExpectations(t)
	})
}

// Helper function to create test project
func createTestProject(projectID uint, name string, slug value.ProjectSlug, ownerID uint) *model.Project {
	// Set resource limits matching MVP policy
	cpuLimit := uint32(1000)
	memoryLimit := uint32(2048)
	diskLimit := uint32(2048)
	trafficLimit := uint32(1048576) // MVP: 1TB

	limits, err := value.NewResourceLimits(cpuLimit, memoryLimit, diskLimit, trafficLimit)
	if err != nil {
		panic(err)
	}

	project, err := model.NewProject(name, slug, ownerID, *limits, nil)
	if err != nil {
		// Handle error - shouldn't happen with valid test data
		panic(err)
	}
	project.SetProjectID(projectID)

	return project
}
