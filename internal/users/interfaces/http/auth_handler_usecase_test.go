package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/users/application/mocks"
	"github.com/swm-launchpad/web-console-backend/internal/users/application/usecase"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/repository"
)

func TestAuthHandler_Register_WithUseCase(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("성공: 정상적인 사용자 등록", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)
		registerUseCase := usecase.NewRegisterUserUseCase(mockRepo, mockAuthService)
		handler := NewAuthHandler(registerUseCase, nil)

		reqBody := RegisterRequest{
			Username: "newuser",
			Password: "StrongPass123!",
			Email:    "newuser@example.com",
			Name:     "New User",
		}

		hashedPassword := "$2a$10$hashedpassword"
		token := "test-token-123"

		// Set expectations
		mockRepo.On("ExistsByUsername", mock.Anything, reqBody.Username).Return(false, nil)
		mockRepo.On("ExistsByEmail", mock.Anything, reqBody.Email).Return(false, nil)
		mockAuthService.On("HashPassword", reqBody.Password).Return(hashedPassword, nil)
		mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(user *model.User) bool {
			return user.Username == reqBody.Username &&
				user.PasswordHash == hashedPassword &&
				user.Email != nil && *user.Email == reqBody.Email &&
				user.Name != nil && *user.Name == reqBody.Name &&
				user.Status == model.UserStatusActive
		})).Return(nil).Run(func(args mock.Arguments) {
			user := args.Get(1).(*model.User)
			user.UserID = 123
		})
		mockAuthService.On("GenerateToken", mock.Anything, uint(123)).Return(token, nil)

		// Act
		router := gin.New()
		router.POST("/auth/register", handler.Register)

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusCreated, w.Code)

		var response RegisterResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, uint(123), response.UserID)
		assert.Equal(t, token, response.Token)
		assert.Equal(t, "User registered successfully", response.Message)

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 이미 존재하는 사용자명", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)
		registerUseCase := usecase.NewRegisterUserUseCase(mockRepo, mockAuthService)
		handler := NewAuthHandler(registerUseCase, nil)

		reqBody := RegisterRequest{
			Username: "existinguser",
			Password: "StrongPass123!",
			Email:    "new@example.com",
			Name:     "New User",
		}

		// Set expectations
		mockRepo.On("ExistsByUsername", mock.Anything, reqBody.Username).Return(true, nil)

		// Act
		router := gin.New()
		router.POST("/auth/register", handler.Register)

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusConflict, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response["error"], "already exists")

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 이미 존재하는 이메일", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)
		registerUseCase := usecase.NewRegisterUserUseCase(mockRepo, mockAuthService)
		handler := NewAuthHandler(registerUseCase, nil)

		reqBody := RegisterRequest{
			Username: "newuser",
			Password: "StrongPass123!",
			Email:    "existing@example.com",
			Name:     "New User",
		}

		// Set expectations
		mockRepo.On("ExistsByUsername", mock.Anything, reqBody.Username).Return(false, nil)
		mockRepo.On("ExistsByEmail", mock.Anything, reqBody.Email).Return(true, nil)

		// Act
		router := gin.New()
		router.POST("/auth/register", handler.Register)

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusConflict, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response["error"], "already exists")

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 비밀번호 해싱 오류", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)
		registerUseCase := usecase.NewRegisterUserUseCase(mockRepo, mockAuthService)
		handler := NewAuthHandler(registerUseCase, nil)

		reqBody := RegisterRequest{
			Username: "newuser",
			Password: "StrongPass123!",
			Email:    "new@example.com",
			Name:     "New User",
		}

		// Set expectations
		mockRepo.On("ExistsByUsername", mock.Anything, reqBody.Username).Return(false, nil)
		mockRepo.On("ExistsByEmail", mock.Anything, reqBody.Email).Return(false, nil)
		mockAuthService.On("HashPassword", reqBody.Password).Return("", errors.New("hashing failed"))

		// Act
		router := gin.New()
		router.POST("/auth/register", handler.Register)

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response["error"], "hashing failed")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 저장 오류", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)
		registerUseCase := usecase.NewRegisterUserUseCase(mockRepo, mockAuthService)
		handler := NewAuthHandler(registerUseCase, nil)

		reqBody := RegisterRequest{
			Username: "newuser",
			Password: "StrongPass123!",
			Email:    "new@example.com",
			Name:     "New User",
		}

		hashedPassword := "$2a$10$hashedpassword"

		// Set expectations
		mockRepo.On("ExistsByUsername", mock.Anything, reqBody.Username).Return(false, nil)
		mockRepo.On("ExistsByEmail", mock.Anything, reqBody.Email).Return(false, nil)
		mockAuthService.On("HashPassword", reqBody.Password).Return(hashedPassword, nil)
		mockRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("database error"))

		// Act
		router := gin.New()
		router.POST("/auth/register", handler.Register)

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response["error"], "database error")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})
}

func TestAuthHandler_Login_WithUseCase(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("성공: 정상적인 로그인", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)
		loginUseCase := usecase.NewLoginUserUseCase(mockRepo, mockAuthService)
		handler := NewAuthHandler(nil, loginUseCase)

		reqBody := LoginRequest{
			Username: "testuser",
			Password: "CorrectPass123!",
		}

		hashedPassword := "$2a$10$hashedpassword"
		token := "login-token-456"

		user := &model.User{
			UserID:       456,
			Username:     "testuser",
			PasswordHash: hashedPassword,
			Email:        stringPtr("test@example.com"),
			Name:         stringPtr("Test User"),
			Status:       model.UserStatusActive,
		}

		// Set expectations
		mockRepo.On("FindByUsername", mock.Anything, reqBody.Username).Return(user, nil)
		mockAuthService.On("VerifyPassword", hashedPassword, reqBody.Password).Return(nil)
		mockAuthService.On("GenerateToken", mock.Anything, uint(456)).Return(token, nil)

		// Act
		router := gin.New()
		router.POST("/auth/login", handler.Login)

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var response LoginResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, uint(456), response.UserID)
		assert.Equal(t, token, response.Token)
		assert.Equal(t, "testuser", response.Username)
		assert.Equal(t, "test@example.com", response.Email)
		assert.Equal(t, "Test User", response.Name)
		assert.Equal(t, "Login successful", response.Message)

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 사용자를 찾을 수 없음", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)
		loginUseCase := usecase.NewLoginUserUseCase(mockRepo, mockAuthService)
		handler := NewAuthHandler(nil, loginUseCase)

		reqBody := LoginRequest{
			Username: "nonexistent",
			Password: "SomePassword123",
		}

		// Set expectations
		mockRepo.On("FindByUsername", mock.Anything, reqBody.Username).Return(nil, repository.ErrUserNotFound)

		// Act
		router := gin.New()
		router.POST("/auth/login", handler.Login)

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response["error"], "invalid credentials")

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 잘못된 비밀번호", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)
		loginUseCase := usecase.NewLoginUserUseCase(mockRepo, mockAuthService)
		handler := NewAuthHandler(nil, loginUseCase)

		reqBody := LoginRequest{
			Username: "testuser",
			Password: "WrongPassword",
		}

		hashedPassword := "$2a$10$hashedpassword"

		user := &model.User{
			UserID:       456,
			Username:     "testuser",
			PasswordHash: hashedPassword,
			Status:       model.UserStatusActive,
		}

		// Set expectations
		mockRepo.On("FindByUsername", mock.Anything, reqBody.Username).Return(user, nil)
		mockAuthService.On("VerifyPassword", hashedPassword, reqBody.Password).Return(errors.New("invalid credentials"))

		// Act
		router := gin.New()
		router.POST("/auth/login", handler.Login)

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response["error"], "invalid credentials")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 비활성 사용자", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)
		loginUseCase := usecase.NewLoginUserUseCase(mockRepo, mockAuthService)
		handler := NewAuthHandler(nil, loginUseCase)

		reqBody := LoginRequest{
			Username: "inactiveuser",
			Password: "CorrectPassword123",
		}

		hashedPassword := "$2a$10$hashedpassword"

		user := &model.User{
			UserID:       789,
			Username:     "inactiveuser",
			PasswordHash: hashedPassword,
			Status:       model.UserStatusInactive,
		}

		// Set expectations - Note: VerifyPassword not called due to early return for inactive user
		mockRepo.On("FindByUsername", mock.Anything, reqBody.Username).Return(user, nil)

		// Act
		router := gin.New()
		router.POST("/auth/login", handler.Login)

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response["error"], "user is not active")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 토큰 생성 오류", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)
		loginUseCase := usecase.NewLoginUserUseCase(mockRepo, mockAuthService)
		handler := NewAuthHandler(nil, loginUseCase)

		reqBody := LoginRequest{
			Username: "testuser",
			Password: "CorrectPass123!",
		}

		hashedPassword := "$2a$10$hashedpassword"

		user := &model.User{
			UserID:       456,
			Username:     "testuser",
			PasswordHash: hashedPassword,
			Status:       model.UserStatusActive,
		}

		// Set expectations
		mockRepo.On("FindByUsername", mock.Anything, reqBody.Username).Return(user, nil)
		mockAuthService.On("VerifyPassword", hashedPassword, reqBody.Password).Return(nil)
		mockAuthService.On("GenerateToken", mock.Anything, uint(456)).Return("", errors.New("token generation failed"))

		// Act
		router := gin.New()
		router.POST("/auth/login", handler.Login)

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response["error"], "failed to generate token")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}