package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {

	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestAuthHandler_Register_WithUseCase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	txManager := db.NewStubTxManager()

	t.Run("성공: 유효한 입력으로 사용자 등록", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		registerUseCase := application.NewRegisterUserUseCase(mockAuthService, txManager)
		handler := NewAuthHandler(registerUseCase, nil)

		createdAt := time.Now()
		name := "Test User"
		user := &model.User{
			UserID:    1,
			Username:  "testuser",
			Email:     "test@example.com",
			Name:      &name,
			Status:    model.UserStatusActive,
			CreatedAt: createdAt,
		}

		mockAuthService.On("RegisterUser", mock.Anything, "testuser", "password123", "test@example.com", &name).
			Return(user, "jwt_token", nil)

		router := gin.New()
		router.POST("/auth/register", handler.Register)

		body := `{"username":"testuser","password":"password123","email":"test@example.com","name":"Test User"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["user_id"])
		assert.Equal(t, "jwt_token", data["token"])
		assert.Equal(t, "User registered successfully", data["message"])

		mockAuthService.AssertExpectations(t)
	})

	t.Run("성공: name 없이 등록", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		registerUseCase := application.NewRegisterUserUseCase(mockAuthService, txManager)
		handler := NewAuthHandler(registerUseCase, nil)

		createdAt := time.Now()
		user := &model.User{
			UserID:    2,
			Username:  "testuser2",
			Email:     "test2@example.com",
			Name:      nil,
			Status:    model.UserStatusActive,
			CreatedAt: createdAt,
		}

		mockAuthService.On("RegisterUser", mock.Anything, "testuser2", "password123", "test2@example.com", (*string)(nil)).
			Return(user, "jwt_token", nil)

		router := gin.New()
		router.POST("/auth/register", handler.Register)

		body := `{"username":"testuser2","password":"password123","email":"test2@example.com"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(2), data["user_id"])
		assert.Equal(t, "jwt_token", data["token"])

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 중복된 username", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		registerUseCase := application.NewRegisterUserUseCase(mockAuthService, txManager)
		handler := NewAuthHandler(registerUseCase, nil)

		name := "Test User"
		mockAuthService.On("RegisterUser", mock.Anything, "existinguser", "password123", "test@example.com", &name).
			Return((*model.User)(nil), "", usererrors.ErrUsernameExists)

		router := gin.New()
		router.POST("/auth/register", handler.Register)

		body := `{"username":"existinguser","password":"password123","email":"test@example.com","name":"Test User"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "USERNAME_EXISTS", errorData["code"])
		assert.Contains(t, errorData["message"], "already exists")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 중복된 email", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		registerUseCase := application.NewRegisterUserUseCase(mockAuthService, txManager)
		handler := NewAuthHandler(registerUseCase, nil)

		name := "Test User"
		mockAuthService.On("RegisterUser", mock.Anything, "testuser", "password123", "existing@example.com", &name).
			Return((*model.User)(nil), "", usererrors.ErrEmailExists)

		router := gin.New()
		router.POST("/auth/register", handler.Register)

		body := `{"username":"testuser","password":"password123","email":"existing@example.com","name":"Test User"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "EMAIL_EXISTS", errorData["code"])
		assert.Contains(t, errorData["message"], "already exists")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 잘못된 요청 형식", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		registerUseCase := application.NewRegisterUserUseCase(mockAuthService, txManager)
		handler := NewAuthHandler(registerUseCase, nil)

		router := gin.New()
		router.POST("/auth/register", handler.Register)

		body := `{invalid json}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "VALIDATION_FAILED", errorData["code"])
		assert.Equal(t, "Validation failed", errorData["message"])

		mockAuthService.AssertNotCalled(t, "RegisterUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestAuthHandler_Login_WithUseCase(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("성공: 유효한 입력으로 로그인", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		loginUseCase := application.NewLoginUserUseCase(mockAuthService)
		handler := NewAuthHandler(nil, loginUseCase)

		name := "Test User"
		user := &model.User{
			UserID:   1,
			Username: "testuser",
			Email:    "test@example.com",
			Name:     &name,
			Status:   model.UserStatusActive,
		}

		mockAuthService.On("AuthenticateUser", context.Background(), "testuser", "password123").
			Return(user, "jwt_token", nil)

		router := gin.New()
		router.POST("/auth/login", handler.Login)

		body := `{"username":"testuser","password":"password123"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["user_id"])
		assert.Equal(t, "jwt_token", data["token"])
		assert.Equal(t, "testuser", data["username"])
		assert.Equal(t, "test@example.com", data["email"])
		assert.Equal(t, "Test User", data["name"])
		assert.Equal(t, "Login successful", data["message"])

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 잘못된 자격증명", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		loginUseCase := application.NewLoginUserUseCase(mockAuthService)
		handler := NewAuthHandler(nil, loginUseCase)

		mockAuthService.On("AuthenticateUser", context.Background(), "testuser", "wrongpassword").
			Return((*model.User)(nil), "", service.ErrInvalidCredentials)

		router := gin.New()
		router.POST("/auth/login", handler.Login)

		body := `{"username":"testuser","password":"wrongpassword"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_CREDENTIALS", errorData["code"])
		assert.Contains(t, errorData["message"], "invalid credentials")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 요청 형식 오류", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		loginUseCase := application.NewLoginUserUseCase(mockAuthService)
		handler := NewAuthHandler(nil, loginUseCase)

		router := gin.New()
		router.POST("/auth/login", handler.Login)

		body := `{invalid json}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "VALIDATION_FAILED", errorData["code"])
		assert.Equal(t, "Validation failed", errorData["message"])

		mockAuthService.AssertNotCalled(t, "AuthenticateUser")
	})
}
