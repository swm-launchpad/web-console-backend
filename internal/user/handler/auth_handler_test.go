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
	"github.com/swm-launchpad/web-console-backend/internal/common/email"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model/token"
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
		mockTokenService := new(service.MockTokenService)
		mockEmailService := new(email.MockService)
		testLogger := logger.NewForTest()
		registerUseCase := application.NewRegisterUserUseCase(mockAuthService, mockTokenService, mockEmailService, txManager, testLogger)
		handler := NewAuthHandler(registerUseCase, nil, testLogger)

		createdAt := time.Now()
		nickname := "testnick"
		user := &model.User{
			UserID:    1,
			Email:     "test@example.com",
			Nickname:  nickname,
			Status:    model.UserStatusPending,
			CreatedAt: createdAt,
		}
		verificationToken := &token.VerificationToken{
			TokenID:   1,
			UserID:    1,
			Token:     "verification_token_123",
			TokenType: token.TokenTypeEmailVerification,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
		}

		mockAuthService.On("RegisterUser", mock.Anything, "test@example.com", "password123", nickname).
			Return(user, "jwt_token", nil)
		mockTokenService.On("CreateEmailVerificationToken", mock.Anything, uint(1)).
			Return(verificationToken, nil)
		mockEmailService.On("SendVerificationEmail", mock.Anything, "test@example.com", nickname, verificationToken.Token).
			Return(nil)

		router := gin.New()
		router.POST("/auth/register", handler.Register)

		body := `{"email":"test@example.com","password":"password123","nickname":"testnick"}`
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
		mockTokenService.AssertExpectations(t)
		mockEmailService.AssertExpectations(t)
	})

	t.Run("실패: 중복된 email", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		mockTokenService := new(service.MockTokenService)
		mockEmailService := new(email.MockService)
		testLogger := logger.NewForTest()
		registerUseCase := application.NewRegisterUserUseCase(mockAuthService, mockTokenService, mockEmailService, txManager, testLogger)
		handler := NewAuthHandler(registerUseCase, nil, testLogger)

		nickname := "testnick"
		mockAuthService.On("RegisterUser", mock.Anything, "existing@example.com", "password123", nickname).
			Return((*model.User)(nil), "", usererrors.ErrEmailExists)

		router := gin.New()
		router.POST("/auth/register", handler.Register)

		body := `{"email":"existing@example.com","password":"password123","nickname":"testnick"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

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
		mockTokenService := new(service.MockTokenService)
		mockEmailService := new(email.MockService)
		testLogger := logger.NewForTest()
		registerUseCase := application.NewRegisterUserUseCase(mockAuthService, mockTokenService, mockEmailService, txManager, testLogger)
		handler := NewAuthHandler(registerUseCase, nil, testLogger)

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
		testLogger := logger.NewForTest()
		loginUseCase := application.NewLoginUserUseCase(mockAuthService, testLogger)
		handler := NewAuthHandler(nil, loginUseCase, testLogger)

		nickname := "testnick"
		user := &model.User{
			UserID:   1,
			Email:    "test@example.com",
			Nickname: nickname,
			Status:   model.UserStatusActive,
		}

		mockAuthService.On("AuthenticateUser", context.Background(), "test@example.com", "password123").
			Return(user, "jwt_token", nil)

		router := gin.New()
		router.POST("/auth/login", handler.Login)

		body := `{"email":"test@example.com","password":"password123"}`
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
		assert.Equal(t, "test@example.com", data["email"])
		assert.Equal(t, nickname, data["nickname"])
		assert.Equal(t, "Login successful", data["message"])

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 잘못된 자격증명", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		loginUseCase := application.NewLoginUserUseCase(mockAuthService, testLogger)
		handler := NewAuthHandler(nil, loginUseCase, testLogger)

		mockAuthService.On("AuthenticateUser", context.Background(), "test@example.com", "wrongpassword").
			Return((*model.User)(nil), "", service.ErrInvalidCredentials)

		router := gin.New()
		router.POST("/auth/login", handler.Login)

		body := `{"email":"test@example.com","password":"wrongpassword"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_CREDENTIALS", errorData["code"])
		assert.Contains(t, errorData["message"], "Invalid credentials")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 요청 형식 오류", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		loginUseCase := application.NewLoginUserUseCase(mockAuthService, testLogger)
		handler := NewAuthHandler(nil, loginUseCase, testLogger)

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
