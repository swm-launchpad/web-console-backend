package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/project/application"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	projectmodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

func TestDeploymentHandler_GetDeployment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	t.Run("Success - Get deployment status", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)
		mockProjectService := new(service.MockProjectService)

		// Create use cases
		getDeploymentUseCase := application.NewGetDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		refreshDeploymentUseCase := &application.RefreshDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
			mockProjectService,
		)

		userID := uint(1)
		projectID := uint(1)

		// Create mock deployment
		d := deployment.NewDeployment(projectID)
		d.SetDeploymentID(100)
		eventID := "test-event-123"
		runName := "test-run-123"
		_ = d.InitTektonInfo(&eventID, &runName)
		summary := "Deployment running"
		startedAt := time.Now()
		_ = d.UpdateRunningStatus(&summary, &startedAt)

		// Mock expectations
		// Mock GetProjectBySlug
		mockProject := &projectmodel.Project{}
		mockProject.SetProjectID(projectID)
		mockProjectService.On("GetProjectBySlug", mock.Anything, "p20250118120000abc12345").Return(mockProject, nil)

		mockPermissionService.On("CanUserAccessProject", ctx, userID, projectID).Return(nil)
		mockDeployService.On("GetDeploymentStatus", ctx, projectID).Return(d, nil)

		// Setup router
		router := gin.New()
		router.GET("/api/v1/projects/:slug/deployments/latest", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/projects/p20250118120000abc12345/deployments/latest", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)
		mockPermissionService.AssertExpectations(t)
		mockDeployService.AssertExpectations(t)
	})

	t.Run("Fail - Permission denied (masked as not found)", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)
		mockProjectService := new(service.MockProjectService)

		// Create use cases
		getDeploymentUseCase := application.NewGetDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		refreshDeploymentUseCase := &application.RefreshDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
			mockProjectService,
		)

		userID := uint(1)
		projectID := uint(1)

		// Mock expectations - permission denied
		// Mock GetProjectBySlug
		mockProject := &projectmodel.Project{}
		mockProject.SetProjectID(projectID)
		mockProjectService.On("GetProjectBySlug", mock.Anything, "p20250118120000abc12345").Return(mockProject, nil)

		mockPermissionService.On("CanUserAccessProject", ctx, userID, projectID).Return(projecterrors.ErrPermissionDenied)

		// Setup router
		router := gin.New()
		router.GET("/api/v1/projects/:slug/deployments/latest", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/projects/p20250118120000abc12345/deployments/latest", nil)
		router.ServeHTTP(w, req)

		// Assertions - should return not found instead of permission denied to prevent information disclosure
		assert.Equal(t, http.StatusNotFound, w.Code)
		mockPermissionService.AssertExpectations(t)
	})

	t.Run("Fail - Deployment not found", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)
		mockProjectService := new(service.MockProjectService)

		// Create use cases
		getDeploymentUseCase := application.NewGetDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		refreshDeploymentUseCase := &application.RefreshDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
			mockProjectService,
		)

		userID := uint(1)
		projectID := uint(1)

		// Mock expectations
		// Mock GetProjectBySlug
		mockProject := &projectmodel.Project{}
		mockProject.SetProjectID(projectID)
		mockProjectService.On("GetProjectBySlug", mock.Anything, "p20250118120000abc12345").Return(mockProject, nil)

		mockPermissionService.On("CanUserAccessProject", ctx, userID, projectID).Return(nil)
		mockDeployService.On("GetDeploymentStatus", ctx, projectID).Return(nil, projecterrors.ErrDeploymentNotFound)

		// Setup router
		router := gin.New()
		router.GET("/api/v1/projects/:slug/deployments/latest", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/projects/p20250118120000abc12345/deployments/latest", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusNotFound, w.Code)
		mockPermissionService.AssertExpectations(t)
		mockDeployService.AssertExpectations(t)
	})

	t.Run("Fail - Unauthorized (no user in context)", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)
		mockProjectService := new(service.MockProjectService)

		// Create use cases
		getDeploymentUseCase := application.NewGetDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		refreshDeploymentUseCase := &application.RefreshDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
			mockProjectService,
		)

		// Setup router - no user in context
		router := gin.New()
		router.GET("/api/v1/projects/:slug/deployments/latest", handler.GetDeployment)

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/projects/p20250118120000abc12345/deployments/latest", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Fail - Invalid project ID", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)
		mockProjectService := new(service.MockProjectService)

		// Mock GetProjectBySlug to return error for invalid slug
		mockProjectService.On("GetProjectBySlug", mock.Anything, "invalid").Return(nil, projecterrors.ErrSlugInvalidFormat)

		// Create use cases
		getDeploymentUseCase := application.NewGetDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		refreshDeploymentUseCase := &application.RefreshDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
			mockProjectService,
		)

		userID := uint(1)

		// Setup router
		router := gin.New()
		router.GET("/api/v1/projects/:slug/deployments/latest", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetDeployment(c)
		})

		// Make request with invalid ID
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/projects/invalid/deployments/latest", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockProjectService.AssertExpectations(t)
	})
}

func TestDeploymentHandler_RefreshDeployment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	t.Run("Success - Refresh deployment status", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)
		mockProjectService := new(service.MockProjectService)

		// Create use cases
		refreshDeploymentUseCase := application.NewRefreshDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		getDeploymentUseCase := &application.GetDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
			mockProjectService,
		)

		userID := uint(1)
		projectID := uint(1)

		// Create mock deployment
		d := deployment.NewDeployment(projectID)
		d.SetDeploymentID(100)
		eventID := "test-event-123"
		runName := "test-run-123"
		_ = d.InitTektonInfo(&eventID, &runName)
		summary := "Deployment running"
		startedAt := time.Now()
		_ = d.UpdateRunningStatus(&summary, &startedAt)

		// Mock expectations
		// Mock GetProjectBySlug
		mockProject := &projectmodel.Project{}
		mockProject.SetProjectID(projectID)
		mockProjectService.On("GetProjectBySlug", mock.Anything, "p20250118120000abc12345").Return(mockProject, nil)

		mockPermissionService.On("CanUserModifyProject", ctx, userID, projectID).Return(nil)
		mockDeployService.On("RefreshActiveDeployment", ctx, projectID).Return(d, nil)

		// Setup router
		router := gin.New()
		router.POST("/api/v1/projects/:slug/deployments/refresh", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.RefreshDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/projects/p20250118120000abc12345/deployments/refresh", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)
		mockPermissionService.AssertExpectations(t)
		mockDeployService.AssertExpectations(t)
	})

	t.Run("Fail - Permission denied (requires modify permission)", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)
		mockProjectService := new(service.MockProjectService)

		// Create use cases
		refreshDeploymentUseCase := application.NewRefreshDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		getDeploymentUseCase := &application.GetDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
			mockProjectService,
		)

		userID := uint(1)
		projectID := uint(1)

		// Mock expectations - permission denied (needs modify, not just access)
		// Mock GetProjectBySlug
		mockProject := &projectmodel.Project{}
		mockProject.SetProjectID(projectID)
		mockProjectService.On("GetProjectBySlug", mock.Anything, "p20250118120000abc12345").Return(mockProject, nil)

		mockPermissionService.On("CanUserModifyProject", ctx, userID, projectID).Return(projecterrors.ErrPermissionDenied)

		// Setup router
		router := gin.New()
		router.POST("/api/v1/projects/:slug/deployments/refresh", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.RefreshDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/projects/p20250118120000abc12345/deployments/refresh", nil)
		router.ServeHTTP(w, req)

		// Assertions - should return not found instead of permission denied
		assert.Equal(t, http.StatusNotFound, w.Code)
		mockPermissionService.AssertExpectations(t)
	})

	t.Run("Fail - No active deployment", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)
		mockProjectService := new(service.MockProjectService)

		// Create use cases
		refreshDeploymentUseCase := application.NewRefreshDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		getDeploymentUseCase := &application.GetDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
			mockProjectService,
		)

		userID := uint(1)
		projectID := uint(1)

		// Mock expectations
		// Mock GetProjectBySlug
		mockProject := &projectmodel.Project{}
		mockProject.SetProjectID(projectID)
		mockProjectService.On("GetProjectBySlug", mock.Anything, "p20250118120000abc12345").Return(mockProject, nil)

		mockPermissionService.On("CanUserModifyProject", ctx, userID, projectID).Return(nil)
		mockDeployService.On("RefreshActiveDeployment", ctx, projectID).Return(nil, projecterrors.ErrDeploymentNotFound)

		// Setup router
		router := gin.New()
		router.POST("/api/v1/projects/:slug/deployments/refresh", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.RefreshDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/projects/p20250118120000abc12345/deployments/refresh", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusNotFound, w.Code)
		mockPermissionService.AssertExpectations(t)
		mockDeployService.AssertExpectations(t)
	})

	t.Run("Fail - Unauthorized (no user in context)", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)
		mockProjectService := new(service.MockProjectService)

		// Create use cases
		refreshDeploymentUseCase := application.NewRefreshDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		getDeploymentUseCase := &application.GetDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
			mockProjectService,
		)

		// Setup router - no user in context
		router := gin.New()
		router.POST("/api/v1/projects/:slug/deployments/refresh", handler.RefreshDeployment)

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/projects/p20250118120000abc12345/deployments/refresh", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Fail - Invalid project ID", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)
		mockProjectService := new(service.MockProjectService)

		// Mock GetProjectBySlug to return error for invalid slug
		mockProjectService.On("GetProjectBySlug", mock.Anything, "invalid").Return(nil, projecterrors.ErrSlugInvalidFormat)

		// Create use cases
		refreshDeploymentUseCase := application.NewRefreshDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		getDeploymentUseCase := &application.GetDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
			mockProjectService,
		)

		userID := uint(1)

		// Setup router
		router := gin.New()
		router.POST("/api/v1/projects/:slug/deployments/refresh", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.RefreshDeployment(c)
		})

		// Make request with invalid ID
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/projects/invalid/deployments/refresh", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockProjectService.AssertExpectations(t)
	})
}

func TestDeploymentHandler_PermissionDifference(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	t.Run("GetDeployment uses access permission (read-only)", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)
		mockProjectService := new(service.MockProjectService)

		// Create use cases
		getDeploymentUseCase := application.NewGetDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		refreshDeploymentUseCase := &application.RefreshDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
			mockProjectService,
		)

		userID := uint(1)
		projectID := uint(1)

		d := deployment.NewDeployment(projectID)
		d.SetDeploymentID(100)

		// Mock expectations - uses CanUserAccessProject
		// Mock GetProjectBySlug
		mockProject := &projectmodel.Project{}
		mockProject.SetProjectID(projectID)
		mockProjectService.On("GetProjectBySlug", mock.Anything, "p20250118120000abc12345").Return(mockProject, nil)

		mockPermissionService.On("CanUserAccessProject", ctx, userID, projectID).Return(nil)
		mockDeployService.On("GetDeploymentStatus", ctx, projectID).Return(d, nil)

		// Setup router
		router := gin.New()
		router.GET("/api/v1/projects/:slug/deployments/latest", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/projects/p20250118120000abc12345/deployments/latest", nil)
		router.ServeHTTP(w, req)

		// Verify CanUserAccessProject was called (not CanUserModifyProject)
		mockPermissionService.AssertCalled(t, "CanUserAccessProject", mock.Anything, userID, projectID)
		mockPermissionService.AssertNotCalled(t, "CanUserModifyProject", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("RefreshDeployment uses modify permission (write operation)", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)
		mockProjectService := new(service.MockProjectService)

		// Create use cases
		refreshDeploymentUseCase := application.NewRefreshDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		getDeploymentUseCase := &application.GetDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
			mockProjectService,
		)

		userID := uint(1)
		projectID := uint(1)

		d := deployment.NewDeployment(projectID)
		d.SetDeploymentID(100)

		// Mock expectations - uses CanUserModifyProject
		// Mock GetProjectBySlug
		mockProject := &projectmodel.Project{}
		mockProject.SetProjectID(projectID)
		mockProjectService.On("GetProjectBySlug", mock.Anything, "p20250118120000abc12345").Return(mockProject, nil)

		mockPermissionService.On("CanUserModifyProject", ctx, userID, projectID).Return(nil)
		mockDeployService.On("RefreshActiveDeployment", ctx, projectID).Return(d, nil)

		// Setup router
		router := gin.New()
		router.POST("/api/v1/projects/:slug/deployments/refresh", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.RefreshDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/projects/p20250118120000abc12345/deployments/refresh", nil)
		router.ServeHTTP(w, req)

		// Verify CanUserModifyProject was called (not CanUserAccessProject)
		mockPermissionService.AssertCalled(t, "CanUserModifyProject", mock.Anything, userID, projectID)
		mockPermissionService.AssertNotCalled(t, "CanUserAccessProject", mock.Anything, mock.Anything, mock.Anything)
	})
}
