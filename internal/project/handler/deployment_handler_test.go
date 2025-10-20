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
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

func TestDeploymentHandler_GetDeployment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	t.Run("Success - Get deployment status", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)

		// Create use cases
		getDeploymentUseCase := application.NewGetDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		refreshDeploymentUseCase := &application.RefreshDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
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
		mockPermissionService.On("CanUserAccessProject", ctx, userID, projectID).Return(nil)
		mockDeployService.On("GetDeploymentStatus", ctx, projectID).Return(d, nil)

		// Setup router
		router := gin.New()
		router.GET("/api/v1/projects/:id/deployments/latest", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/projects/1/deployments/latest", nil)
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

		// Create use cases
		getDeploymentUseCase := application.NewGetDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		refreshDeploymentUseCase := &application.RefreshDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
		)

		userID := uint(1)
		projectID := uint(1)

		// Mock expectations - permission denied
		mockPermissionService.On("CanUserAccessProject", ctx, userID, projectID).Return(projecterrors.ErrPermissionDenied)

		// Setup router
		router := gin.New()
		router.GET("/api/v1/projects/:id/deployments/latest", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/projects/1/deployments/latest", nil)
		router.ServeHTTP(w, req)

		// Assertions - should return not found instead of permission denied to prevent information disclosure
		assert.Equal(t, http.StatusNotFound, w.Code)
		mockPermissionService.AssertExpectations(t)
	})

	t.Run("Fail - Deployment not found", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)

		// Create use cases
		getDeploymentUseCase := application.NewGetDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		refreshDeploymentUseCase := &application.RefreshDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
		)

		userID := uint(1)
		projectID := uint(1)

		// Mock expectations
		mockPermissionService.On("CanUserAccessProject", ctx, userID, projectID).Return(nil)
		mockDeployService.On("GetDeploymentStatus", ctx, projectID).Return(nil, projecterrors.ErrDeploymentNotFound)

		// Setup router
		router := gin.New()
		router.GET("/api/v1/projects/:id/deployments/latest", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/projects/1/deployments/latest", nil)
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

		// Create use cases
		getDeploymentUseCase := application.NewGetDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		refreshDeploymentUseCase := &application.RefreshDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
		)

		// Setup router - no user in context
		router := gin.New()
		router.GET("/api/v1/projects/:id/deployments/latest", handler.GetDeployment)

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/projects/1/deployments/latest", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Fail - Invalid project ID", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)

		// Create use cases
		getDeploymentUseCase := application.NewGetDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		refreshDeploymentUseCase := &application.RefreshDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
		)

		userID := uint(1)

		// Setup router
		router := gin.New()
		router.GET("/api/v1/projects/:id/deployments/latest", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetDeployment(c)
		})

		// Make request with invalid ID
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/projects/invalid/deployments/latest", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestDeploymentHandler_RefreshDeployment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	t.Run("Success - Refresh deployment status", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)

		// Create use cases
		refreshDeploymentUseCase := application.NewRefreshDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		getDeploymentUseCase := &application.GetDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
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
		mockPermissionService.On("CanUserModifyProject", ctx, userID, projectID).Return(nil)
		mockDeployService.On("RefreshActiveDeployment", ctx, projectID).Return(d, nil)

		// Setup router
		router := gin.New()
		router.POST("/api/v1/projects/:id/deployments/refresh", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.RefreshDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/projects/1/deployments/refresh", nil)
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

		// Create use cases
		refreshDeploymentUseCase := application.NewRefreshDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		getDeploymentUseCase := &application.GetDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
		)

		userID := uint(1)
		projectID := uint(1)

		// Mock expectations - permission denied (needs modify, not just access)
		mockPermissionService.On("CanUserModifyProject", ctx, userID, projectID).Return(projecterrors.ErrPermissionDenied)

		// Setup router
		router := gin.New()
		router.POST("/api/v1/projects/:id/deployments/refresh", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.RefreshDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/projects/1/deployments/refresh", nil)
		router.ServeHTTP(w, req)

		// Assertions - should return not found instead of permission denied
		assert.Equal(t, http.StatusNotFound, w.Code)
		mockPermissionService.AssertExpectations(t)
	})

	t.Run("Fail - No active deployment", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)

		// Create use cases
		refreshDeploymentUseCase := application.NewRefreshDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		getDeploymentUseCase := &application.GetDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
		)

		userID := uint(1)
		projectID := uint(1)

		// Mock expectations
		mockPermissionService.On("CanUserModifyProject", ctx, userID, projectID).Return(nil)
		mockDeployService.On("RefreshActiveDeployment", ctx, projectID).Return(nil, projecterrors.ErrDeploymentNotFound)

		// Setup router
		router := gin.New()
		router.POST("/api/v1/projects/:id/deployments/refresh", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.RefreshDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/projects/1/deployments/refresh", nil)
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

		// Create use cases
		refreshDeploymentUseCase := application.NewRefreshDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		getDeploymentUseCase := &application.GetDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
		)

		// Setup router - no user in context
		router := gin.New()
		router.POST("/api/v1/projects/:id/deployments/refresh", handler.RefreshDeployment)

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/projects/1/deployments/refresh", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Fail - Invalid project ID", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)

		// Create use cases
		refreshDeploymentUseCase := application.NewRefreshDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		getDeploymentUseCase := &application.GetDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
		)

		userID := uint(1)

		// Setup router
		router := gin.New()
		router.POST("/api/v1/projects/:id/deployments/refresh", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.RefreshDeployment(c)
		})

		// Make request with invalid ID
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/projects/invalid/deployments/refresh", nil)
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestDeploymentHandler_PermissionDifference(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	t.Run("GetDeployment uses access permission (read-only)", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)

		// Create use cases
		getDeploymentUseCase := application.NewGetDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		refreshDeploymentUseCase := &application.RefreshDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
		)

		userID := uint(1)
		projectID := uint(1)

		d := deployment.NewDeployment(projectID)
		d.SetDeploymentID(100)

		// Mock expectations - uses CanUserAccessProject
		mockPermissionService.On("CanUserAccessProject", ctx, userID, projectID).Return(nil)
		mockDeployService.On("GetDeploymentStatus", ctx, projectID).Return(d, nil)

		// Setup router
		router := gin.New()
		router.GET("/api/v1/projects/:id/deployments/latest", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/projects/1/deployments/latest", nil)
		router.ServeHTTP(w, req)

		// Verify CanUserAccessProject was called (not CanUserModifyProject)
		mockPermissionService.AssertCalled(t, "CanUserAccessProject", mock.Anything, userID, projectID)
		mockPermissionService.AssertNotCalled(t, "CanUserModifyProject", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("RefreshDeployment uses modify permission (write operation)", func(t *testing.T) {
		// Setup mocks
		mockDeployService := new(service.MockDeployService)
		mockPermissionService := new(service.MockPermissionService)

		// Create use cases
		refreshDeploymentUseCase := application.NewRefreshDeploymentUseCase(mockDeployService)
		deployProjectUseCase := &application.DeployProjectUseCase{}
		getDeploymentUseCase := &application.GetDeploymentUseCase{}

		handler := NewDeploymentHandler(
			deployProjectUseCase,
			getDeploymentUseCase,
			refreshDeploymentUseCase,
			mockPermissionService,
		)

		userID := uint(1)
		projectID := uint(1)

		d := deployment.NewDeployment(projectID)
		d.SetDeploymentID(100)

		// Mock expectations - uses CanUserModifyProject
		mockPermissionService.On("CanUserModifyProject", ctx, userID, projectID).Return(nil)
		mockDeployService.On("RefreshActiveDeployment", ctx, projectID).Return(d, nil)

		// Setup router
		router := gin.New()
		router.POST("/api/v1/projects/:id/deployments/refresh", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.RefreshDeployment(c)
		})

		// Make request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/projects/1/deployments/refresh", nil)
		router.ServeHTTP(w, req)

		// Verify CanUserModifyProject was called (not CanUserAccessProject)
		mockPermissionService.AssertCalled(t, "CanUserModifyProject", mock.Anything, userID, projectID)
		mockPermissionService.AssertNotCalled(t, "CanUserAccessProject", mock.Anything, mock.Anything, mock.Anything)
	})
}
