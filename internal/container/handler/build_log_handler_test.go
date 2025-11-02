package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/application"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	containermodel "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
	containerservice "github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
)

func TestBuildLogHandler_GetBuildLogHistory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	// Setup mocks
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockPermissionService := new(infrastructure.MockPermissionService)
	mockContainerService := new(containerservice.MockContainerService)
	testLogger := logger.NewForTest()

	// Create real use case with mocked dependencies
	getBuildLogHistoryUC := application.NewGetBuildLogHistoryUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		mockPermissionService,
		testLogger,
	)

	userID := uint(1)
	containerID := uint(10)

	// Mock permission check - user has access
	mockPermissionService.On("CanUserAccessContainer", ctx, userID, containerID).Return(nil)

	handler := NewBuildLogHandler(
		nil, // createBuildLogTokenUC not needed
		nil, // streamBuildLogsUC not needed
		getBuildLogHistoryUC,
		mockContainerService,
		nil, // jwtUtil not needed
		testLogger,
	)

	containerSlug := "test-container-slug"

	// Create test container
	slug, _ := value.NewContainerSlug(containerSlug)
	gitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "main", nil)
	resourceLimits, _ := value.NewResourceLimits(nil, nil)
	testContainer, _ := containermodel.NewContainer(
		1,          // projectID
		"test-app", // name
		slug,
		gitConfig,
		resourceLimits,
		nil, // templateID
		nil, // templateConfig
		nil, // githubInstallationID
	)
	testContainer.SetContainerID(containerID)

	// Create completed build
	pipelineRunName := "image-build-push-run-abc123"
	startedAt := time.Now().Add(-1 * time.Hour)
	finishedAt := time.Now().Add(-30 * time.Minute)

	completedBuild, _ := build_history.ReconstructBuildHistory(
		1,
		containerID,
		build_history.BuildHistoryStatusSuccess,
		nil,
		stringPtr("event-123"),
		&pipelineRunName,
		nil,
		time.Now().Add(-2*time.Hour),
		&startedAt,
		&finishedAt,
	)

	// Mock expectations
	mockContainerService.On("GetContainerBySlug", ctx, containerSlug).
		Return(testContainer, nil)

	mockBuildHistoryRepo.On("FindLatestByContainerID", ctx, containerID).
		Return(completedBuild, nil)

	mockLogData := io.NopCloser(strings.NewReader(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
	mockLokiClient.On("QueryPipelineRunLogsHTTP", ctx, pipelineRunName, []string{"ecr-repository-check"}, startedAt, finishedAt).
		Return(mockLogData, nil)

	// Setup router
	router := gin.New()
	router.GET("/containers/:slug/build-logs/history", func(c *gin.Context) {
		c.Set(auth.ContextKeyUserID, userID)
		handler.GetBuildLogHistory(c)
	})

	// Execute request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/containers/"+containerSlug+"/build-logs/history", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")

	mockContainerService.AssertExpectations(t)
	mockBuildHistoryRepo.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
}

func TestBuildLogHandler_GetBuildLogHistory_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mocks
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockPermissionService := new(infrastructure.MockPermissionService)
	mockContainerService := new(containerservice.MockContainerService)
	testLogger := logger.NewForTest()

	getBuildLogHistoryUC := application.NewGetBuildLogHistoryUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		mockPermissionService,
		testLogger,
	)

	handler := NewBuildLogHandler(
		nil,
		nil,
		getBuildLogHistoryUC,
		mockContainerService,
		nil,
		testLogger,
	)

	containerSlug := "test-container-slug"

	// Setup router WITHOUT setting userID in context
	router := gin.New()
	router.GET("/containers/:slug/build-logs/history", handler.GetBuildLogHistory)

	// Execute request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/containers/"+containerSlug+"/build-logs/history", nil)
	router.ServeHTTP(w, req)

	// Assert - should return 401
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// No services should be called
	mockContainerService.AssertNotCalled(t, "GetContainerBySlug")
	mockBuildHistoryRepo.AssertNotCalled(t, "FindLatestByContainerID")
	mockLokiClient.AssertNotCalled(t, "QueryPipelineRunLogsHTTP")
}

func TestBuildLogHandler_GetBuildLogHistory_MissingSlug(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mocks
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockPermissionService := new(infrastructure.MockPermissionService)
	mockContainerService := new(containerservice.MockContainerService)
	testLogger := logger.NewForTest()

	getBuildLogHistoryUC := application.NewGetBuildLogHistoryUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		mockPermissionService,
		testLogger,
	)

	handler := NewBuildLogHandler(
		nil,
		nil,
		getBuildLogHistoryUC,
		mockContainerService,
		nil,
		testLogger,
	)

	userID := uint(1)

	// Setup router with empty slug
	router := gin.New()
	router.GET("/containers/:slug/build-logs/history", func(c *gin.Context) {
		c.Set(auth.ContextKeyUserID, userID)
		// Manually set empty slug
		c.Params = gin.Params{{Key: "slug", Value: ""}}
		handler.GetBuildLogHistory(c)
	})

	// Execute request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/containers//build-logs/history", nil)
	router.ServeHTTP(w, req)

	// Assert - should return 401 (treats as unauthorized)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// No services should be called
	mockContainerService.AssertNotCalled(t, "GetContainerBySlug")
	mockBuildHistoryRepo.AssertNotCalled(t, "FindLatestByContainerID")
	mockLokiClient.AssertNotCalled(t, "QueryPipelineRunLogsHTTP")
}

func TestBuildLogHandler_GetBuildLogHistory_ContainerNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	// Setup mocks
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockPermissionService := new(infrastructure.MockPermissionService)
	mockContainerService := new(containerservice.MockContainerService)
	testLogger := logger.NewForTest()

	getBuildLogHistoryUC := application.NewGetBuildLogHistoryUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		mockPermissionService,
		testLogger,
	)

	handler := NewBuildLogHandler(
		nil,
		nil,
		getBuildLogHistoryUC,
		mockContainerService,
		nil,
		testLogger,
	)

	userID := uint(1)
	containerSlug := "nonexistent-container"

	// Mock GetContainerBySlug - container not found
	mockContainerService.On("GetContainerBySlug", ctx, containerSlug).
		Return(nil, containererrors.ErrContainerNotFound)

	// Setup router
	router := gin.New()
	router.GET("/containers/:slug/build-logs/history", func(c *gin.Context) {
		c.Set(auth.ContextKeyUserID, userID)
		handler.GetBuildLogHistory(c)
	})

	// Execute request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/containers/"+containerSlug+"/build-logs/history", nil)
	router.ServeHTTP(w, req)

	// Assert - should return 404
	assert.Equal(t, http.StatusNotFound, w.Code)

	mockContainerService.AssertExpectations(t)
	// BuildHistory and Loki should not be called
	mockBuildHistoryRepo.AssertNotCalled(t, "FindLatestByContainerID")
	mockLokiClient.AssertNotCalled(t, "QueryPipelineRunLogsHTTP")
}

func TestBuildLogHandler_GetBuildLogHistory_NoBuildHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	// Setup mocks
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockPermissionService := new(infrastructure.MockPermissionService)
	mockContainerService := new(containerservice.MockContainerService)
	testLogger := logger.NewForTest()

	getBuildLogHistoryUC := application.NewGetBuildLogHistoryUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		mockPermissionService,
		testLogger,
	)

	handler := NewBuildLogHandler(
		nil,
		nil,
		getBuildLogHistoryUC,
		mockContainerService,
		nil,
		testLogger,
	)

	userID := uint(1)
	containerSlug := "test-container-slug"
	containerID := uint(10)

	// Mock permission check - user has access
	mockPermissionService.On("CanUserAccessContainer", ctx, userID, containerID).Return(nil)

	// Create test container
	slug, _ := value.NewContainerSlug(containerSlug)
	gitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "main", nil)
	resourceLimits, _ := value.NewResourceLimits(nil, nil)
	testContainer, _ := containermodel.NewContainer(
		1,          // projectID
		"test-app", // name
		slug,
		gitConfig,
		resourceLimits,
		nil, // templateID
		nil, // templateConfig
		nil, // githubInstallationID
	)
	testContainer.SetContainerID(containerID)

	// Mock expectations
	mockContainerService.On("GetContainerBySlug", ctx, containerSlug).
		Return(testContainer, nil)

	mockBuildHistoryRepo.On("FindLatestByContainerID", ctx, containerID).
		Return(nil, projecterrors.ErrBuildHistoryNotFound)

	// Setup router
	router := gin.New()
	router.GET("/containers/:slug/build-logs/history", func(c *gin.Context) {
		c.Set(auth.ContextKeyUserID, userID)
		handler.GetBuildLogHistory(c)
	})

	// Execute request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/containers/"+containerSlug+"/build-logs/history", nil)
	router.ServeHTTP(w, req)

	// Assert - should return 404
	assert.Equal(t, http.StatusNotFound, w.Code)

	mockContainerService.AssertExpectations(t)
	mockBuildHistoryRepo.AssertExpectations(t)
	// Loki should not be called
	mockLokiClient.AssertNotCalled(t, "QueryPipelineRunLogsHTTP")
}

func TestBuildLogHandler_GetBuildLogHistory_LokiFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	// Setup mocks
	mockBuildHistoryRepo := new(repository.MockBuildHistoryRepository)
	mockLokiClient := new(infrastructure.MockLokiClient)
	mockPermissionService := new(infrastructure.MockPermissionService)
	mockContainerService := new(containerservice.MockContainerService)
	testLogger := logger.NewForTest()

	getBuildLogHistoryUC := application.NewGetBuildLogHistoryUseCase(
		mockBuildHistoryRepo,
		mockLokiClient,
		mockPermissionService,
		testLogger,
	)

	handler := NewBuildLogHandler(
		nil,
		nil,
		getBuildLogHistoryUC,
		mockContainerService,
		nil,
		testLogger,
	)

	userID := uint(1)
	containerSlug := "test-container-slug"
	containerID := uint(10)

	// Mock permission check - user has access
	mockPermissionService.On("CanUserAccessContainer", ctx, userID, containerID).Return(nil)

	// Create test container
	slug, _ := value.NewContainerSlug(containerSlug)
	gitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "main", nil)
	resourceLimits, _ := value.NewResourceLimits(nil, nil)
	testContainer, _ := containermodel.NewContainer(
		1,          // projectID
		"test-app", // name
		slug,
		gitConfig,
		resourceLimits,
		nil, // templateID
		nil, // templateConfig
		nil, // githubInstallationID
	)
	testContainer.SetContainerID(containerID)

	// Create completed build
	pipelineRunName := "image-build-push-run-fail"
	startedAt := time.Now().Add(-1 * time.Hour)
	finishedAt := time.Now().Add(-30 * time.Minute)

	completedBuild, _ := build_history.ReconstructBuildHistory(
		1,
		containerID,
		build_history.BuildHistoryStatusSuccess,
		nil,
		stringPtr("event-fail"),
		&pipelineRunName,
		nil,
		time.Now().Add(-2*time.Hour),
		&startedAt,
		&finishedAt,
	)

	// Mock expectations
	mockContainerService.On("GetContainerBySlug", ctx, containerSlug).
		Return(testContainer, nil)

	mockBuildHistoryRepo.On("FindLatestByContainerID", ctx, containerID).
		Return(completedBuild, nil)

	// Mock Loki failure
	lokiError := errors.New("loki connection failed")
	mockLokiClient.On("QueryPipelineRunLogsHTTP", ctx, pipelineRunName, []string{"ecr-repository-check"}, startedAt, finishedAt).
		Return(nil, lokiError)

	// Setup router
	router := gin.New()
	router.GET("/containers/:slug/build-logs/history", func(c *gin.Context) {
		c.Set(auth.ContextKeyUserID, userID)
		handler.GetBuildLogHistory(c)
	})

	// Execute request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/containers/"+containerSlug+"/build-logs/history", nil)
	router.ServeHTTP(w, req)

	// Assert - should return 500 (internal server error)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mockContainerService.AssertExpectations(t)
	mockBuildHistoryRepo.AssertExpectations(t)
	mockLokiClient.AssertExpectations(t)
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
