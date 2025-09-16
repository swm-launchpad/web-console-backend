package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/users/application/mocks"
	"github.com/swm-launchpad/web-console-backend/internal/users/application/usecase"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/repository"
)

func TestUserHandler_GetCurrentUser_WithUseCase(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("성공: 인증된 사용자 프로필 조회", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		getUserUseCase := usecase.NewGetUserUseCase(mockRepo)
		handler := NewUserHandler(getUserUseCase)

		userID := uint(123)
		createdAt := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

		user := &model.User{
			UserID:       userID,
			Username:     "currentuser",
			Email:        stringPtrHelper("current@example.com"),
			Name:         stringPtrHelper("Current User"),
			Phone:        stringPtrHelper("123-456-7890"),
			Organization: stringPtrHelper("Test Org"),
			Status:       model.UserStatusActive,
			CreatedAt:    createdAt,
		}

		// Set expectations
		mockRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

		// Act
		router := gin.New()
		router.GET("/users/me", func(c *gin.Context) {
			c.Set("userID", userID) // Simulate authenticated user
			handler.GetCurrentUser(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var response UserResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, userID, response.UserID)
		assert.Equal(t, "currentuser", response.Username)
		assert.Equal(t, "current@example.com", response.Email)
		assert.Equal(t, "Current User", response.Name)
		assert.Equal(t, "123-456-7890", response.Phone)
		assert.Equal(t, "Test Org", response.Organization)
		assert.Equal(t, "active", response.Status)
		assert.Equal(t, "2024-01-15T10:30:45Z", response.CreatedAt)

		mockRepo.AssertExpectations(t)
	})

	t.Run("성공: 선택적 필드가 비어있는 경우", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		getUserUseCase := usecase.NewGetUserUseCase(mockRepo)
		handler := NewUserHandler(getUserUseCase)

		userID := uint(456)
		createdAt := time.Date(2024, 2, 20, 15, 45, 30, 0, time.UTC)

		user := &model.User{
			UserID:       userID,
			Username:     "minimaluser",
			Email:        nil, // Empty optional field
			Name:         nil, // Empty optional field
			Phone:        nil, // Empty optional field
			Organization: nil, // Empty optional field
			Status:       model.UserStatusActive,
			CreatedAt:    createdAt,
		}

		// Set expectations
		mockRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

		// Act
		router := gin.New()
		router.GET("/users/me", func(c *gin.Context) {
			c.Set("userID", userID)
			handler.GetCurrentUser(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var response UserResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, userID, response.UserID)
		assert.Equal(t, "minimaluser", response.Username)
		assert.Empty(t, response.Email)
		assert.Empty(t, response.Name)
		assert.Empty(t, response.Phone)
		assert.Empty(t, response.Organization)
		assert.Equal(t, "active", response.Status)
		assert.Equal(t, "2024-02-20T15:45:30Z", response.CreatedAt)

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 사용자를 찾을 수 없음", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		getUserUseCase := usecase.NewGetUserUseCase(mockRepo)
		handler := NewUserHandler(getUserUseCase)

		userID := uint(999)

		// Set expectations
		mockRepo.On("FindByID", mock.Anything, userID).Return(nil, repository.ErrUserNotFound)

		// Act
		router := gin.New()
		router.GET("/users/me", func(c *gin.Context) {
			c.Set("userID", userID)
			handler.GetCurrentUser(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "User not found", response["error"])

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 오류", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		getUserUseCase := usecase.NewGetUserUseCase(mockRepo)
		handler := NewUserHandler(getUserUseCase)

		userID := uint(123)

		// Set expectations
		mockRepo.On("FindByID", mock.Anything, userID).Return(nil, errors.New("database error"))

		// Act
		router := gin.New()
		router.GET("/users/me", func(c *gin.Context) {
			c.Set("userID", userID)
			handler.GetCurrentUser(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Failed to fetch user profile", response["error"])

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 타입 어설션 실패", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		getUserUseCase := usecase.NewGetUserUseCase(mockRepo)
		handler := NewUserHandler(getUserUseCase)

		// Act
		router := gin.New()
		router.GET("/users/me", func(c *gin.Context) {
			c.Set("userID", "invalid-type") // Wrong type for userID
			handler.GetCurrentUser(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		w := httptest.NewRecorder()

		// This should panic due to type assertion failure
		assert.Panics(t, func() {
			router.ServeHTTP(w, req)
		})
	})
}

func TestUserHandler_GetUserByID_WithUseCase(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("성공: ID로 사용자 조회", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		getUserUseCase := usecase.NewGetUserUseCase(mockRepo)
		handler := NewUserHandler(getUserUseCase)

		userID := uint(789)
		createdAt := time.Date(2024, 3, 10, 8, 15, 30, 0, time.UTC)

		user := &model.User{
			UserID:       userID,
			Username:     "targetuser",
			Email:        stringPtrHelper("target@example.com"),
			Name:         stringPtrHelper("Target User"),
			Phone:        stringPtrHelper("987-654-3210"),
			Organization: stringPtrHelper("Another Org"),
			Status:       model.UserStatusInactive,
			CreatedAt:    createdAt,
		}

		// Set expectations
		mockRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

		// Act
		router := gin.New()
		router.GET("/users/:id", handler.GetUserByID)

		req := httptest.NewRequest(http.MethodGet, "/users/789", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var response UserResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, userID, response.UserID)
		assert.Equal(t, "targetuser", response.Username)
		assert.Equal(t, "target@example.com", response.Email)
		assert.Equal(t, "Target User", response.Name)
		assert.Equal(t, "987-654-3210", response.Phone)
		assert.Equal(t, "Another Org", response.Organization)
		assert.Equal(t, "inactive", response.Status)
		assert.Equal(t, "2024-03-10T08:15:30Z", response.CreatedAt)

		mockRepo.AssertExpectations(t)
	})

	t.Run("성공: 유효한 범위의 다양한 ID", func(t *testing.T) {
		testCases := []struct {
			name   string
			userID uint
		}{
			{"최소값 ID", 1},
			{"중간값 ID", 50000},
			{"큰 값 ID", 4294967295}, // Max uint32
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Arrange
				mockRepo := new(mocks.MockUserRepository)
				getUserUseCase := usecase.NewGetUserUseCase(mockRepo)
				handler := NewUserHandler(getUserUseCase)

				user := &model.User{
					UserID:    tc.userID,
					Username:  "testuser",
					Email:     stringPtrHelper("test@example.com"),
					Name:      stringPtrHelper("Test User"),
					Status:    model.UserStatusActive,
					CreatedAt: time.Now(),
				}

				// Set expectations
				mockRepo.On("FindByID", mock.Anything, tc.userID).Return(user, nil)

				// Act
				router := gin.New()
				router.GET("/users/:id", handler.GetUserByID)

				req := httptest.NewRequest(http.MethodGet, "/users/"+strconv.FormatUint(uint64(tc.userID), 10), nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				// Assert
				assert.Equal(t, http.StatusOK, w.Code)

				var response UserResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tc.userID, response.UserID)

				mockRepo.AssertExpectations(t)
			})
		}
	})

	t.Run("실패: 사용자를 찾을 수 없음", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		getUserUseCase := usecase.NewGetUserUseCase(mockRepo)
		handler := NewUserHandler(getUserUseCase)

		userID := uint(404)

		// Set expectations
		mockRepo.On("FindByID", mock.Anything, userID).Return(nil, repository.ErrUserNotFound)

		// Act
		router := gin.New()
		router.GET("/users/:id", handler.GetUserByID)

		req := httptest.NewRequest(http.MethodGet, "/users/404", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "User not found", response["error"])

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 접근 오류", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		getUserUseCase := usecase.NewGetUserUseCase(mockRepo)
		handler := NewUserHandler(getUserUseCase)

		userID := uint(500)

		// Set expectations
		mockRepo.On("FindByID", mock.Anything, userID).Return(nil, errors.New("connection timeout"))

		// Act
		router := gin.New()
		router.GET("/users/:id", handler.GetUserByID)

		req := httptest.NewRequest(http.MethodGet, "/users/500", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Failed to fetch user profile", response["error"])

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 내부 서버 오류", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		getUserUseCase := usecase.NewGetUserUseCase(mockRepo)
		handler := NewUserHandler(getUserUseCase)

		userID := uint(100)

		// Set expectations
		mockRepo.On("FindByID", mock.Anything, userID).Return(nil, errors.New("unexpected error"))

		// Act
		router := gin.New()
		router.GET("/users/:id", handler.GetUserByID)

		req := httptest.NewRequest(http.MethodGet, "/users/100", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Failed to fetch user profile", response["error"])

		mockRepo.AssertExpectations(t)
	})
}

// Helper function to create string pointer
func stringPtrHelper(s string) *string {
	return &s
}