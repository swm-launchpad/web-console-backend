package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

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
		assert.Equal(t, float64(1), response["user_id"])
		assert.Equal(t, "jwt_token", response["token"])
		assert.Equal(t, "User registered successfully", response["message"])

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
		assert.Equal(t, float64(2), response["user_id"])
		assert.Equal(t, "jwt_token", response["token"])

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 중복된 username", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		registerUseCase := application.NewRegisterUserUseCase(mockAuthService, txManager)
		handler := NewAuthHandler(registerUseCase, nil)

		name := "Test User"
		mockAuthService.On("RegisterUser", mock.Anything, "existinguser", "password123", "test@example.com", &name).
			Return((*model.User)(nil), "", errors.New("username already exists"))

		router := gin.New()
		router.POST("/auth/register", handler.Register)

		body := `{"username":"existinguser","password":"password123","email":"test@example.com","name":"Test User"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "already exists")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 중복된 email", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		registerUseCase := application.NewRegisterUserUseCase(mockAuthService, txManager)
		handler := NewAuthHandler(registerUseCase, nil)

		name := "Test User"
		mockAuthService.On("RegisterUser", mock.Anything, "testuser", "password123", "existing@example.com", &name).
			Return((*model.User)(nil), "", errors.New("email already exists"))

		router := gin.New()
		router.POST("/auth/register", handler.Register)

		body := `{"username":"testuser","password":"password123","email":"existing@example.com","name":"Test User"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "already exists")

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

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid request format")

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
		assert.Equal(t, float64(1), response["user_id"])
		assert.Equal(t, "jwt_token", response["token"])
		assert.Equal(t, "testuser", response["username"])
		assert.Equal(t, "test@example.com", response["email"])
		assert.Equal(t, "Test User", response["name"])
		assert.Equal(t, "Login successful", response["message"])

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

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "invalid credentials")

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

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid request format")

		mockAuthService.AssertNotCalled(t, "AuthenticateUser")
	})
}
