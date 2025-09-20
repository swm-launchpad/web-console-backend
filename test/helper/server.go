package helper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/password"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	domainerrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/error"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
	userhttp "github.com/swm-launchpad/web-console-backend/internal/user/handler"
	"github.com/swm-launchpad/web-console-backend/internal/user/infrastructure"
)

// TestServer는 테스트용 HTTP 서버를 제공합니다
type TestServer struct {
	Router *gin.Engine
	DB     *TestDB
}

// SetupTestServer는 테스트용 서버를 설정합니다
func SetupTestServer(t *testing.T) *TestServer {
	t.Helper()

	// Initialize error registry for tests
	initializeTestErrorRegistry()

	// Gin 테스트 모드 설정
	gin.SetMode(gin.TestMode)

	// 테스트 DB 설정
	testDB := SetupTestDB(t)

	// 의존성 초기화
	userRepo := infrastructure.NewUserRepository(testDB.DB)
	jwtUtil := jwt.NewJWTUtil("test-secret")
	passwordUtil := password.NewPasswordUtil()
	txManager := db.NewTxManager(testDB.DB)

	// Service 초기화
	userService := service.NewUserService(userRepo)
	authService := service.NewAuthService(userService, jwtUtil, passwordUtil)

	// UseCase 초기화
	registerUseCase := application.NewRegisterUserUseCase(authService, txManager)
	loginUseCase := application.NewLoginUserUseCase(authService)
	getUserUseCase := application.NewGetUserUseCase(userService)

	// Handler 초기화
	authHandler := userhttp.NewAuthHandler(registerUseCase, loginUseCase)
	userHandler := userhttp.NewUserHandler(getUserUseCase)

	// Router 설정
	router := gin.New()
	router.Use(gin.Recovery())

	// Routes
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
	}

	userGroup := router.Group("/users")
	{
		// 인증 미들웨어를 시뮬레이션하는 테스트용 핸들러
		userGroup.GET("/me", func(c *gin.Context) {
			// 테스트에서 Authorization 헤더로 userID 전달
			if authHeader := c.GetHeader("X-User-ID"); authHeader != "" {
				var userID uint
				if _, err := fmt.Sscanf(authHeader, "%d", &userID); err == nil {
					c.Set(auth.ContextKeyUserID, userID)
				}
			}
			userHandler.GetCurrentUser(c)
		})
		userGroup.GET("/:id", userHandler.GetUserByID)
	}

	return &TestServer{
		Router: router,
		DB:     testDB,
	}
}

// initializeTestErrorRegistry registers all errors for testing
func initializeTestErrorRegistry() {
	// Register auth package errors
	response.RegisterError(auth.ErrInvalidCredentials, auth.CodeInvalidCredentials, http.StatusUnauthorized, "Invalid username or password")
	response.RegisterError(auth.ErrTokenExpired, auth.CodeTokenExpired, http.StatusUnauthorized, "Token has expired")
	response.RegisterError(auth.ErrInvalidToken, auth.CodeInvalidToken, http.StatusUnauthorized, "Invalid or expired token")
	response.RegisterError(auth.ErrUnauthorized, auth.CodeUnauthorized, http.StatusUnauthorized, "User not authenticated")
	response.RegisterError(auth.ErrInvalidRefreshToken, auth.CodeInvalidRefreshToken, http.StatusUnauthorized, "Invalid refresh token")
	response.RegisterError(auth.ErrUserNotActive, auth.CodeUserNotActive, http.StatusForbidden, "User account is not active")
	response.RegisterError(auth.ErrTokenGenerationFailed, auth.CodeTokenGenerationFailed, http.StatusInternalServerError, "Failed to generate authentication token")
	response.RegisterError(auth.ErrPasswordTooWeak, auth.CodePasswordTooWeak, http.StatusBadRequest, "Password does not meet security requirements")
	response.RegisterError(auth.ErrPasswordMismatch, auth.CodePasswordMismatch, http.StatusBadRequest, "Password does not match")
	response.RegisterError(auth.ErrMissingAuthHeader, auth.CodeMissingAuthHeader, http.StatusUnauthorized, "Authorization header is required")
	response.RegisterError(auth.ErrInvalidAuthFormat, auth.CodeInvalidAuthFormat, http.StatusUnauthorized, "Invalid authorization header format")
	response.RegisterError(auth.ErrMissingToken, auth.CodeMissingToken, http.StatusUnauthorized, "Token is required")

	// Register user domain errors
	response.RegisterError(domainerrors.ErrUserNotFound, domainerrors.CodeUserNotFound, http.StatusNotFound, "User not found")
	response.RegisterError(domainerrors.ErrUserAlreadyExists, domainerrors.CodeUserAlreadyExists, http.StatusConflict, "User already exists")
	response.RegisterError(domainerrors.ErrInvalidUserData, domainerrors.CodeInvalidUserData, http.StatusBadRequest, "Invalid user data")
	response.RegisterError(domainerrors.ErrUserNotActive, domainerrors.CodeUserNotActive, http.StatusForbidden, "User is not active")
	response.RegisterError(domainerrors.ErrCannotActivateDeletedUser, domainerrors.CodeCannotActivateDeletedUser, http.StatusBadRequest, "Cannot activate deleted user")
	response.RegisterError(domainerrors.ErrCannotDeleteUser, domainerrors.CodeCannotDeleteUser, http.StatusBadRequest, "Cannot delete user")

	// Register authentication errors
	response.RegisterError(domainerrors.ErrInvalidCredentials, domainerrors.CodeInvalidCredentials, http.StatusUnauthorized, "Invalid username or password")
	response.RegisterError(domainerrors.ErrWeakPassword, domainerrors.CodeWeakPassword, http.StatusBadRequest, "Password does not meet security requirements")
	response.RegisterError(domainerrors.ErrInvalidEmail, domainerrors.CodeInvalidEmail, http.StatusBadRequest, "Invalid email format")

	// Register validation errors
	response.RegisterError(domainerrors.ErrUsernameRequired, domainerrors.CodeUsernameRequired, http.StatusBadRequest, "Username is required")
	response.RegisterError(domainerrors.ErrPasswordRequired, domainerrors.CodePasswordRequired, http.StatusBadRequest, "Password is required")
	response.RegisterError(domainerrors.ErrEmailRequired, domainerrors.CodeEmailRequired, http.StatusBadRequest, "Email is required")
	response.RegisterError(domainerrors.ErrUsernameTooShort, domainerrors.CodeUsernameTooShort, http.StatusBadRequest, "Username must be at least 3 characters long")
	response.RegisterError(domainerrors.ErrInvalidUserID, domainerrors.CodeInvalidUserID, http.StatusBadRequest, "Invalid user ID")
	response.RegisterError(domainerrors.ErrPasswordEmpty, domainerrors.CodePasswordEmpty, http.StatusBadRequest, "Password cannot be empty")

	// Register duplicate errors
	response.RegisterError(domainerrors.ErrUsernameExists, domainerrors.CodeUsernameExists, http.StatusConflict, "Username already exists")
	response.RegisterError(domainerrors.ErrEmailExists, domainerrors.CodeEmailExists, http.StatusConflict, "Email already exists")

	// Register common validation errors
	response.RegisterError(response.ErrValidationFailed, response.CodeValidationFailed, http.StatusBadRequest, "Validation failed")
	response.RegisterError(response.ErrInvalidFormat, response.CodeInvalidFormat, http.StatusBadRequest, "Invalid format")
	response.RegisterError(response.ErrMissingField, response.CodeMissingField, http.StatusBadRequest, "Required field is missing")
}

// Cleanup는 테스트 서버를 정리합니다
func (ts *TestServer) Cleanup() {
	if ts.DB != nil {
		ts.DB.Cleanup()
	}
}

// MakeRequest는 HTTP 요청을 만들고 응답을 반환합니다
func (ts *TestServer) MakeRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	w := httptest.NewRecorder()
	ts.Router.ServeHTTP(w, req)
	return w
}

// MakeAuthenticatedRequest는 인증된 요청을 만듭니다
func (ts *TestServer) MakeAuthenticatedRequest(method, path string, body interface{}, userID uint) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	// 테스트용 userID 헤더 추가
	req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))

	w := httptest.NewRecorder()
	ts.Router.ServeHTTP(w, req)
	return w
}

// RegisterUser는 테스트용 사용자를 등록하고 토큰을 반환합니다
func (ts *TestServer) RegisterUser(t *testing.T, username, password, email string) (uint, string) {
	t.Helper()

	reqBody := map[string]string{
		"username": username,
		"password": password,
		"email":    email,
	}

	w := ts.MakeRequest("POST", "/auth/register", reqBody)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to register user: %s", w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Extract data from new response format
	data := response["data"].(map[string]interface{})
	userID := uint(data["user_id"].(float64))
	token := data["token"].(string)

	return userID, token
}

// LoginUser는 테스트용 로그인을 수행하고 토큰을 반환합니다
func (ts *TestServer) LoginUser(t *testing.T, username, password string) (uint, string) {
	t.Helper()

	reqBody := map[string]string{
		"username": username,
		"password": password,
	}

	w := ts.MakeRequest("POST", "/auth/login", reqBody)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to login user: %s", w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Extract data from new response format
	data := response["data"].(map[string]interface{})
	userID := uint(data["user_id"].(float64))
	token := data["token"].(string)

	return userID, token
}
