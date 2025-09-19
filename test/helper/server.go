package helper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/password"
	"github.com/swm-launchpad/web-console-backend/internal/users/application/usecase"
	"github.com/swm-launchpad/web-console-backend/internal/users/infrastructure/persistence"
	userhttp "github.com/swm-launchpad/web-console-backend/internal/users/interfaces/http"
)

// TestServer는 테스트용 HTTP 서버를 제공합니다
type TestServer struct {
	Router *gin.Engine
	DB     *TestDB
}

// testAuthService는 테스트용 AuthService 구현입니다
type testAuthService struct {
	jwtService      *jwt.Service
	passwordService *password.Service
}

func (a *testAuthService) GenerateToken(ctx context.Context, userID uint) (string, error) {
	return a.jwtService.GenerateToken(ctx, userID)
}

func (a *testAuthService) ValidateToken(ctx context.Context, token string) (uint, error) {
	return a.jwtService.ValidateToken(ctx, token)
}

func (a *testAuthService) HashPassword(password string) (string, error) {
	return a.passwordService.HashPassword(password)
}

func (a *testAuthService) VerifyPassword(password, hash string) error {
	return a.passwordService.VerifyPassword(password, hash)
}

// SetupTestServer는 테스트용 서버를 설정합니다
func SetupTestServer(t *testing.T) *TestServer {
	t.Helper()

	// Gin 테스트 모드 설정
	gin.SetMode(gin.TestMode)

	// 테스트 DB 설정
	db := SetupTestDB(t)

	// 의존성 초기화
	userRepo := persistence.NewUserRepository(db.DB)
	jwtService := jwt.NewService("test-secret")
	passwordService := password.NewService()

	// AuthService 인터페이스를 구현하는 구조체
	authService := &testAuthService{
		jwtService:      jwtService,
		passwordService: passwordService,
	}

	// UseCase 초기화
	registerUseCase := usecase.NewRegisterUserUseCase(userRepo, authService)
	loginUseCase := usecase.NewLoginUserUseCase(userRepo, authService)
	getUserUseCase := usecase.NewGetUserUseCase(userRepo)

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