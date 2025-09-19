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
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
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

	// Gin 테스트 모드 설정
	gin.SetMode(gin.TestMode)

	// 테스트 DB 설정
	db := SetupTestDB(t)

	// 의존성 초기화
	userRepo := infrastructure.NewUserRepository(db.DB)
	jwtUtil := jwt.NewJWTUtil("test-secret")
	passwordUtil := password.NewPasswordUtil()

	// Service 초기화
	userService := service.NewUserService(userRepo)
	authService := service.NewAuthService(userService, jwtUtil, passwordUtil)

	// UseCase 초기화
	registerUseCase := application.NewRegisterUserUseCase(authService)
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
		DB:     db,
	}
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

	userID := uint(response["user_id"].(float64))
	token := response["token"].(string)

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

	userID := uint(response["user_id"].(float64))
	token := response["token"].(string)

	return userID, token
}
