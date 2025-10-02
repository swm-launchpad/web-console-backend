package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/project/application"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

func TestVolumeHandler_AddVolume(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	t.Run("성공: 볼륨 추가", func(t *testing.T) {
		// Setup mocks
		mockVolumeService := new(service.MockVolumeService)
		mockPermissionService := new(service.MockPermissionService)
		txManager := db.NewStubTxManager()

		// Create real use cases with mocked dependencies
		addVolumeUseCase := application.NewAddVolumeUseCase(mockVolumeService, txManager)
		getVolumesUseCase := application.NewGetVolumesUseCase(mockVolumeService)
		removeVolumeUseCase := application.NewRemoveVolumeUseCase(mockVolumeService, txManager)

		handler := NewVolumeHandler(
			addVolumeUseCase,
			getVolumesUseCase,
			removeVolumeUseCase,
			mockPermissionService,
		)

		userID := uint(1)
		projectID := uint(1)
		volumeName := "test-volume"
		capacity := uint32(1024)

		// Mock expectations
		mockPermissionService.On("CanUserAddVolume", ctx, userID, projectID).Return(nil)

		volume, _ := model.NewVolume(projectID, volumeName, capacity)
		volume.SetVolumeID(1)
		mockVolumeService.On("CreateVolume", ctx, projectID, volumeName, capacity).Return(volume, nil)

		router := gin.New()
		router.POST("/volumes", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.AddVolume(c)
		})

		reqBody := map[string]interface{}{
			"project_id": projectID,
			"name":       volumeName,
			"capacity":   capacity,
		}
		jsonBody, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/volumes", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["volume_id"])
		assert.Equal(t, volumeName, data["name"])
		assert.Equal(t, float64(capacity), data["capacity"])

		mockVolumeService.AssertExpectations(t)
		mockPermissionService.AssertExpectations(t)
	})

	t.Run("실패: 인증되지 않은 사용자", func(t *testing.T) {
		// Setup mocks
		mockVolumeService := new(service.MockVolumeService)
		mockPermissionService := new(service.MockPermissionService)
		txManager := db.NewStubTxManager()

		// Create real use cases with mocked dependencies
		addVolumeUseCase := application.NewAddVolumeUseCase(mockVolumeService, txManager)
		getVolumesUseCase := application.NewGetVolumesUseCase(mockVolumeService)
		removeVolumeUseCase := application.NewRemoveVolumeUseCase(mockVolumeService, txManager)

		handler := NewVolumeHandler(
			addVolumeUseCase,
			getVolumesUseCase,
			removeVolumeUseCase,
			mockPermissionService,
		)

		router := gin.New()
		router.POST("/volumes", handler.AddVolume)

		reqBody := map[string]interface{}{
			"project_id": 1,
			"name":       "test-volume",
			"capacity":   1024,
		}
		jsonBody, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/volumes", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		mockPermissionService.AssertNotCalled(t, "CanUserAddVolume")
	})

	t.Run("실패: 권한 없음", func(t *testing.T) {
		// Setup mocks
		mockVolumeService := new(service.MockVolumeService)
		mockPermissionService := new(service.MockPermissionService)
		txManager := db.NewStubTxManager()

		// Create real use cases with mocked dependencies
		addVolumeUseCase := application.NewAddVolumeUseCase(mockVolumeService, txManager)
		getVolumesUseCase := application.NewGetVolumesUseCase(mockVolumeService)
		removeVolumeUseCase := application.NewRemoveVolumeUseCase(mockVolumeService, txManager)

		handler := NewVolumeHandler(
			addVolumeUseCase,
			getVolumesUseCase,
			removeVolumeUseCase,
			mockPermissionService,
		)

		userID := uint(1)
		projectID := uint(1)

		// Mock permission denied
		mockPermissionService.On("CanUserAddVolume", ctx, userID, projectID).Return(projecterrors.ErrPermissionDenied)

		router := gin.New()
		router.POST("/volumes", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.AddVolume(c)
		})

		reqBody := map[string]interface{}{
			"project_id": projectID,
			"name":       "test-volume",
			"capacity":   1024,
		}
		jsonBody, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/volumes", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// Return 404 instead of 403 for security (information disclosure prevention)
		assert.Equal(t, http.StatusNotFound, w.Code)

		mockPermissionService.AssertExpectations(t)
		mockVolumeService.AssertNotCalled(t, "AddVolume")
	})
}

func TestVolumeHandler_GetVolumes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	t.Run("성공: 특정 프로젝트의 볼륨 조회", func(t *testing.T) {
		// Setup mocks
		mockVolumeService := new(service.MockVolumeService)
		mockPermissionService := new(service.MockPermissionService)
		txManager := db.NewStubTxManager()

		// Create real use cases with mocked dependencies
		addVolumeUseCase := application.NewAddVolumeUseCase(mockVolumeService, txManager)
		getVolumesUseCase := application.NewGetVolumesUseCase(mockVolumeService)
		removeVolumeUseCase := application.NewRemoveVolumeUseCase(mockVolumeService, txManager)

		handler := NewVolumeHandler(
			addVolumeUseCase,
			getVolumesUseCase,
			removeVolumeUseCase,
			mockPermissionService,
		)

		userID := uint(1)
		projectID := uint(1)

		// Mock expectations
		mockPermissionService.On("CanUserAccessProject", ctx, userID, projectID).Return(nil)

		volume1, _ := model.NewVolume(projectID, "volume1", 1024)
		volume1.SetVolumeID(1)
		volume2, _ := model.NewVolume(projectID, "volume2", 2048)
		volume2.SetVolumeID(2)

		volumes := []*model.Volume{volume1, volume2}
		mockVolumeService.On("ListVolumesByProjectID", ctx, projectID).Return(volumes, nil)

		router := gin.New()
		router.GET("/volumes", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetVolumes(c)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/volumes?project_id="+strconv.Itoa(int(projectID)), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		volumesData := data["volumes"].([]interface{})
		assert.Len(t, volumesData, 2)

		mockVolumeService.AssertExpectations(t)
		mockPermissionService.AssertExpectations(t)
	})

	t.Run("실패: 권한 없음", func(t *testing.T) {
		// Setup mocks
		mockVolumeService := new(service.MockVolumeService)
		mockPermissionService := new(service.MockPermissionService)
		txManager := db.NewStubTxManager()

		// Create real use cases with mocked dependencies
		addVolumeUseCase := application.NewAddVolumeUseCase(mockVolumeService, txManager)
		getVolumesUseCase := application.NewGetVolumesUseCase(mockVolumeService)
		removeVolumeUseCase := application.NewRemoveVolumeUseCase(mockVolumeService, txManager)

		handler := NewVolumeHandler(
			addVolumeUseCase,
			getVolumesUseCase,
			removeVolumeUseCase,
			mockPermissionService,
		)

		userID := uint(1)
		projectID := uint(1)

		// Mock permission denied
		mockPermissionService.On("CanUserAccessProject", ctx, userID, projectID).Return(projecterrors.ErrPermissionDenied)

		router := gin.New()
		router.GET("/volumes", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetVolumes(c)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/volumes?project_id="+strconv.Itoa(int(projectID)), nil)
		router.ServeHTTP(w, req)

		// Return 404 instead of 403 for security (information disclosure prevention)
		assert.Equal(t, http.StatusNotFound, w.Code)

		mockPermissionService.AssertExpectations(t)
		mockVolumeService.AssertNotCalled(t, "ListVolumesByProjectID")
	})
}

func TestVolumeHandler_RemoveVolume(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	t.Run("성공: 볼륨 제거", func(t *testing.T) {
		// Setup mocks
		mockVolumeService := new(service.MockVolumeService)
		mockPermissionService := new(service.MockPermissionService)
		txManager := db.NewStubTxManager()

		// Create real use cases with mocked dependencies
		addVolumeUseCase := application.NewAddVolumeUseCase(mockVolumeService, txManager)
		getVolumesUseCase := application.NewGetVolumesUseCase(mockVolumeService)
		removeVolumeUseCase := application.NewRemoveVolumeUseCase(mockVolumeService, txManager)

		handler := NewVolumeHandler(
			addVolumeUseCase,
			getVolumesUseCase,
			removeVolumeUseCase,
			mockPermissionService,
		)

		userID := uint(1)
		volumeID := uint(1)

		// Mock expectations
		mockPermissionService.On("CanUserRemoveVolume", ctx, userID, volumeID).Return(nil)
		mockVolumeService.On("DeleteVolume", ctx, volumeID).Return(nil)

		router := gin.New()
		router.DELETE("/volumes/:id", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.RemoveVolume(c)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/volumes/"+strconv.Itoa(int(volumeID)), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))

		mockVolumeService.AssertExpectations(t)
		mockPermissionService.AssertExpectations(t)
	})

	t.Run("실패: 인증되지 않은 사용자", func(t *testing.T) {
		// Setup mocks
		mockVolumeService := new(service.MockVolumeService)
		mockPermissionService := new(service.MockPermissionService)
		txManager := db.NewStubTxManager()

		// Create real use cases with mocked dependencies
		addVolumeUseCase := application.NewAddVolumeUseCase(mockVolumeService, txManager)
		getVolumesUseCase := application.NewGetVolumesUseCase(mockVolumeService)
		removeVolumeUseCase := application.NewRemoveVolumeUseCase(mockVolumeService, txManager)

		handler := NewVolumeHandler(
			addVolumeUseCase,
			getVolumesUseCase,
			removeVolumeUseCase,
			mockPermissionService,
		)

		volumeID := uint(1)

		router := gin.New()
		router.DELETE("/volumes/:id", handler.RemoveVolume)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/volumes/"+strconv.Itoa(int(volumeID)), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		mockPermissionService.AssertNotCalled(t, "CanUserRemoveVolume")
	})

	t.Run("실패: 볼륨을 찾을 수 없음", func(t *testing.T) {
		// Setup mocks
		mockVolumeService := new(service.MockVolumeService)
		mockPermissionService := new(service.MockPermissionService)
		txManager := db.NewStubTxManager()

		// Create real use cases with mocked dependencies
		addVolumeUseCase := application.NewAddVolumeUseCase(mockVolumeService, txManager)
		getVolumesUseCase := application.NewGetVolumesUseCase(mockVolumeService)
		removeVolumeUseCase := application.NewRemoveVolumeUseCase(mockVolumeService, txManager)

		handler := NewVolumeHandler(
			addVolumeUseCase,
			getVolumesUseCase,
			removeVolumeUseCase,
			mockPermissionService,
		)

		userID := uint(1)
		volumeID := uint(999)

		// Mock volume not found
		mockPermissionService.On("CanUserRemoveVolume", ctx, userID, volumeID).Return(projecterrors.ErrVolumeNotFound)

		router := gin.New()
		router.DELETE("/volumes/:id", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.RemoveVolume(c)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/volumes/"+strconv.Itoa(int(volumeID)), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		mockPermissionService.AssertExpectations(t)
		mockVolumeService.AssertNotCalled(t, "DeleteVolume")
	})
}
