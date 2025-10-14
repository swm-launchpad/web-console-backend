package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}

// Response types for handler tests
type GetCurrentUserResponse struct {
	UserID       uint   `json:"user_id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Name         string `json:"name,omitempty"`
	Phone        string `json:"phone,omitempty"`
	Organization string `json:"organization,omitempty"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}

type GetUserByIDResponse struct {
	UserID       uint   `json:"user_id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Name         string `json:"name,omitempty"`
	Phone        string `json:"phone,omitempty"`
	Organization string `json:"organization,omitempty"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}

func TestUserHandler_GetCurrentUser_WithUseCase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	t.Run("성공: 인증된 사용자 프로필 조회", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		getUserUseCase := application.NewGetUserUseCase(mockService)
		handler := NewUserHandler(getUserUseCase, nil, nil)

		userID := uint(123)
		createdAt := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

		user := &model.User{
			UserID:       userID,
			Username:     "currentuser",
			Email:        "current@example.com",
			Name:         stringPtr("Current User"),
			Phone:        stringPtr("123-456-7890"),
			Organization: stringPtr("Test Org"),
			Status:       model.UserStatusActive,
			CreatedAt:    createdAt,
		}

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return(user, nil)

		// Act
		router := gin.New()
		router.GET("/user/me", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetCurrentUser(c)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/user/me", nil)
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Check response structure
		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})

		assert.Equal(t, float64(userID), data["user_id"])
		assert.Equal(t, "currentuser", data["username"])
		assert.Equal(t, "current@example.com", data["email"])
		assert.Equal(t, "Current User", data["name"])
		assert.Equal(t, "123-456-7890", data["phone"])
		assert.Equal(t, "Test Org", data["organization"])
		assert.Equal(t, "active", data["status"])
		assert.Equal(t, createdAt.Format("2006-01-02T15:04:05Z"), data["created_at"])

		mockService.AssertExpectations(t)
	})

	t.Run("실패: 사용자를 찾을 수 없음", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		getUserUseCase := application.NewGetUserUseCase(mockService)
		handler := NewUserHandler(getUserUseCase, nil, nil)

		userID := uint(456)

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return((*model.User)(nil), usererrors.ErrUserNotFound)

		// Act
		router := gin.New()
		router.GET("/user/me", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetCurrentUser(c)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/user/me", nil)
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Check error response structure
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "USER_NOT_FOUND", errorData["code"])
		assert.Equal(t, "User not found", errorData["message"])

		mockService.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 오류", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		getUserUseCase := application.NewGetUserUseCase(mockService)
		handler := NewUserHandler(getUserUseCase, nil, nil)

		userID := uint(789)

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return((*model.User)(nil), errors.New("database connection error"))

		// Act
		router := gin.New()
		router.GET("/user/me", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetCurrentUser(c)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/user/me", nil)
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Check error response structure
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "INTERNAL_ERROR", errorData["code"])
		assert.Contains(t, errorData["message"], "database")

		mockService.AssertExpectations(t)
	})

	t.Run("성공: 선택적 필드가 nil인 경우", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		getUserUseCase := application.NewGetUserUseCase(mockService)
		handler := NewUserHandler(getUserUseCase, nil, nil)

		userID := uint(321)
		createdAt := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

		user := &model.User{
			UserID:       userID,
			Username:     "minimaluser",
			Email:        "minimal@example.com",
			Name:         nil,
			Phone:        nil,
			Organization: nil,
			Status:       model.UserStatusPending,
			CreatedAt:    createdAt,
		}

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return(user, nil)

		// Act
		router := gin.New()
		router.GET("/user/me", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetCurrentUser(c)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/user/me", nil)
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Check response structure
		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})

		assert.Equal(t, float64(userID), data["user_id"])
		assert.Equal(t, "minimaluser", data["username"])
		assert.Equal(t, "minimal@example.com", data["email"])
		// Optional fields should be omitted when empty (due to json omitempty tag)
		assert.Nil(t, data["name"])
		assert.Nil(t, data["phone"])
		assert.Nil(t, data["organization"])
		assert.Equal(t, "pending", data["status"])
		assert.Equal(t, createdAt.Format("2006-01-02T15:04:05Z"), data["created_at"])

		mockService.AssertExpectations(t)
	})

	t.Run("실패: userID가 0인 경우", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		getUserUseCase := application.NewGetUserUseCase(mockService)
		handler := NewUserHandler(getUserUseCase, nil, nil)

		userID := uint(0)

		// Set expectations - GetUserByID will return error for invalid userID
		mockService.On("GetUserByID", ctx, userID).Return((*model.User)(nil), errors.New("invalid user ID"))

		// Act
		router := gin.New()
		router.GET("/user/me", func(c *gin.Context) {
			c.Set(auth.ContextKeyUserID, userID)
			handler.GetCurrentUser(c)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/user/me", nil)
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Check error response structure
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "INTERNAL_ERROR", errorData["code"])
		assert.Equal(t, "invalid user ID", errorData["message"].(string))

		mockService.AssertExpectations(t)
	})

	t.Run("실패: userID가 context에 없는 경우", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		getUserUseCase := application.NewGetUserUseCase(mockService)
		handler := NewUserHandler(getUserUseCase, nil, nil)

		// Act
		router := gin.New()
		router.GET("/user/me", handler.GetCurrentUser)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/user/me", nil)
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Check error response structure
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "UNAUTHORIZED", errorData["code"])
		assert.Equal(t, "Unauthorized", errorData["message"])

		// mockService should not be called
		mockService.AssertNotCalled(t, "GetUserByID")
	})
}

func TestUserHandler_GetUserByID_WithUseCase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	t.Run("성공: ID로 사용자 조회", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		getUserUseCase := application.NewGetUserUseCase(mockService)
		handler := NewUserHandler(getUserUseCase, nil, nil)

		userID := uint(100)
		createdAt := time.Date(2024, 2, 20, 14, 25, 30, 0, time.UTC)

		user := &model.User{
			UserID:       userID,
			Username:     "searchuser",
			Email:        "search@example.com",
			Name:         stringPtr("Search User"),
			Phone:        stringPtr("987-654-3210"),
			Organization: stringPtr("Search Corp"),
			Status:       model.UserStatusActive,
			CreatedAt:    createdAt,
		}

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return(user, nil)

		// Act
		router := gin.New()
		router.GET("/users/:id", handler.GetUserByID)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/users/"+strconv.Itoa(int(userID)), nil)
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Check response structure
		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})

		assert.Equal(t, float64(userID), data["user_id"])
		assert.Equal(t, "searchuser", data["username"])
		assert.Equal(t, "search@example.com", data["email"])
		assert.Equal(t, "Search User", data["name"])
		assert.Equal(t, "987-654-3210", data["phone"])
		assert.Equal(t, "Search Corp", data["organization"])
		assert.Equal(t, "active", data["status"])
		assert.Equal(t, createdAt.Format("2006-01-02T15:04:05Z"), data["created_at"])

		mockService.AssertExpectations(t)
	})

	t.Run("실패: 잘못된 ID 형식", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		getUserUseCase := application.NewGetUserUseCase(mockService)
		handler := NewUserHandler(getUserUseCase, nil, nil)

		// Act
		router := gin.New()
		router.GET("/users/:id", handler.GetUserByID)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/users/not-a-number", nil)
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Check error response structure
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_FORMAT", errorData["code"])
		assert.Equal(t, "Invalid format", errorData["message"])

		// mockService should not be called
		mockService.AssertNotCalled(t, "GetUserByID")
	})

	t.Run("실패: 사용자를 찾을 수 없음", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		getUserUseCase := application.NewGetUserUseCase(mockService)
		handler := NewUserHandler(getUserUseCase, nil, nil)

		userID := uint(999)

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return((*model.User)(nil), usererrors.ErrUserNotFound)

		// Act
		router := gin.New()
		router.GET("/users/:id", handler.GetUserByID)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/users/"+strconv.Itoa(int(userID)), nil)
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Check error response structure
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "USER_NOT_FOUND", errorData["code"])
		assert.Equal(t, "User not found", errorData["message"])

		mockService.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 오류", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		getUserUseCase := application.NewGetUserUseCase(mockService)
		handler := NewUserHandler(getUserUseCase, nil, nil)

		userID := uint(200)

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return((*model.User)(nil), errors.New("database error"))

		// Act
		router := gin.New()
		router.GET("/users/:id", handler.GetUserByID)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/users/"+strconv.Itoa(int(userID)), nil)
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Check error response structure
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "INTERNAL_ERROR", errorData["code"])
		assert.Contains(t, errorData["message"], "database")

		mockService.AssertExpectations(t)
	})
}

// Helper function to create string pointer
