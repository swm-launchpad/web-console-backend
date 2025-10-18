package helper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/password"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/email"
	"github.com/swm-launchpad/web-console-backend/internal/common/middleware"
	containerApp "github.com/swm-launchpad/web-console-backend/internal/container/application"
	containerService "github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	containerHTTP "github.com/swm-launchpad/web-console-backend/internal/container/handler"
	containerInfra "github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
	projectApp "github.com/swm-launchpad/web-console-backend/internal/project/application"
	projectService "github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	projectHTTP "github.com/swm-launchpad/web-console-backend/internal/project/handler"
	projectRepo "github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/repository"
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
	testDB := SetupTestDB(t)

	// 의존성 초기화
	userRepo := infrastructure.NewUserRepository(testDB.DB)
	tokenRepo := infrastructure.NewTokenRepository(testDB.DB)
	jwtUtil := jwt.NewJWTUtil("test-secret")
	passwordUtil := password.NewPasswordUtil()
	txManager := db.NewTxManager(testDB.DB)

	// Email Service 초기화 (테스트용 Mock 사용)
	mockEmailService := new(email.MockService)
	// Mock email service to always succeed
	mockEmailService.On("SendVerificationEmail", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockEmailService.On("SendPasswordResetEmail", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// User Service 초기화
	userService := service.NewUserService(userRepo)
	authService := service.NewAuthService(userService, jwtUtil, passwordUtil)
	tokenService := service.NewTokenService(tokenRepo)

	// User UseCase 초기화
	registerUseCase := application.NewRegisterUserUseCase(authService, tokenService, mockEmailService, txManager)
	loginUseCase := application.NewLoginUserUseCase(authService)
	getUserUseCase := application.NewGetUserUseCase(userService)
	updateUserUseCase := application.NewUpdateUserUseCase(userService)
	changePasswordUseCase := application.NewChangePasswordUseCase(userService, authService)

	// Project dependencies
	projectRepository := projectRepo.NewProjectRepository(testDB.DB)
	volumeRepo := projectRepo.NewVolumeRepository(testDB.DB)
	slugService := projectService.NewSlugService(projectRepository)
	volumeSlugService := projectService.NewVolumeSlugService(volumeRepo)
	projectSvc := projectService.NewProjectService(projectRepository, slugService)
	volumeSvc := projectService.NewVolumeService(volumeRepo, projectRepository, volumeSlugService)
	permissionSvc := projectService.NewPermissionService(projectRepository, volumeRepo)

	// Project UseCases
	createProjectUseCase := projectApp.NewCreateProjectUseCase(projectSvc, txManager)
	getProjectUseCase := projectApp.NewGetProjectUseCase(projectSvc, volumeSvc)
	updateProjectUseCase := projectApp.NewUpdateProjectUseCase(projectSvc, txManager)
	deleteProjectUseCase := projectApp.NewDeleteProjectUseCase(projectSvc, volumeSvc, txManager)
	listProjectsUseCase := projectApp.NewListProjectsUseCase(projectSvc)
	addVolumeUseCase := projectApp.NewAddVolumeUseCase(volumeSvc, txManager)
	getVolumesUseCase := projectApp.NewGetVolumesUseCase(volumeSvc)
	removeVolumeUseCase := projectApp.NewRemoveVolumeUseCase(volumeSvc, txManager)

	// Container dependencies
	containerRepo := containerInfra.NewContainerRepository(testDB.DB)
	templateRepo := containerInfra.NewTemplateRepository(testDB.DB)
	containerSlugService := containerService.NewSlugService(containerRepo)
	containerSvc := containerService.NewContainerService(containerRepo, containerSlugService)
	containerPermissionSvc := containerService.NewPermissionService(containerRepo, projectRepository)
	resourceValidationSvc := containerService.NewResourceValidationService(containerRepo, projectRepository)

	// Container UseCases
	createContainerUseCase := containerApp.NewCreateContainerUseCase(containerSvc, containerRepo, containerPermissionSvc, resourceValidationSvc, volumeSvc, txManager)
	getContainerUseCase := containerApp.NewGetContainerUseCase(containerRepo, containerPermissionSvc)
	listContainersUseCase := containerApp.NewListContainersUseCase(containerRepo, containerPermissionSvc)
	updateContainerUseCase := containerApp.NewUpdateContainerUseCase(containerRepo, containerPermissionSvc, resourceValidationSvc, txManager)
	deleteContainerUseCase := containerApp.NewDeleteContainerUseCase(containerRepo, containerPermissionSvc, txManager)
	addEnvVarUseCase := containerApp.NewAddEnvVarUseCase(containerRepo, containerPermissionSvc, txManager)
	updateEnvVarUseCase := containerApp.NewUpdateEnvVarUseCase(containerRepo, containerPermissionSvc, txManager)
	deleteEnvVarUseCase := containerApp.NewDeleteEnvVarUseCase(containerRepo, containerPermissionSvc, txManager)
	addNetworkUseCase := containerApp.NewAddNetworkUseCase(containerRepo, containerPermissionSvc, txManager)
	deleteNetworkUseCase := containerApp.NewDeleteNetworkUseCase(containerRepo, containerPermissionSvc, txManager)
	addSecretUseCase := containerApp.NewAddSecretUseCase(containerRepo, containerPermissionSvc, txManager)
	updateSecretUseCase := containerApp.NewUpdateSecretUseCase(containerRepo, containerPermissionSvc, txManager)
	deleteSecretUseCase := containerApp.NewDeleteSecretUseCase(containerRepo, containerPermissionSvc, txManager)
	addMountUseCase := containerApp.NewAddMountUseCase(containerRepo, containerPermissionSvc, volumeSvc, txManager)
	deleteMountUseCase := containerApp.NewDeleteMountUseCase(containerRepo, containerPermissionSvc, txManager)

	// Template UseCases
	getTemplatesUseCase := containerApp.NewGetTemplatesUseCase(templateRepo)
	getTemplateUseCase := containerApp.NewGetTemplateUseCase(templateRepo)

	// Handler 초기화
	authHandler := userhttp.NewAuthHandler(registerUseCase, loginUseCase)
	userHandler := userhttp.NewUserHandler(getUserUseCase, updateUserUseCase, changePasswordUseCase)
	projectHandler := projectHTTP.NewProjectHandler(
		createProjectUseCase,
		getProjectUseCase,
		updateProjectUseCase,
		deleteProjectUseCase,
		listProjectsUseCase,
		permissionSvc,
		projectSvc,
	)
	volumeHandler := projectHTTP.NewVolumeHandler(
		addVolumeUseCase,
		getVolumesUseCase,
		removeVolumeUseCase,
		permissionSvc,
	)
	containerHandler := containerHTTP.NewContainerHandler(
		createContainerUseCase,
		getContainerUseCase,
		updateContainerUseCase,
		deleteContainerUseCase,
		listContainersUseCase,
		addEnvVarUseCase,
		updateEnvVarUseCase,
		deleteEnvVarUseCase,
		addNetworkUseCase,
		deleteNetworkUseCase,
		addSecretUseCase,
		updateSecretUseCase,
		deleteSecretUseCase,
		addMountUseCase,
		deleteMountUseCase,
	)
	templateHandler := containerHTTP.NewTemplateHandler(
		getTemplatesUseCase,
		getTemplateUseCase,
	)

	// Middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtUtil)

	// Router 설정
	router := gin.New()
	router.Use(gin.Recovery())

	// Routes
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
	}

	// API v1 routes
	v1 := router.Group("/api/v1")

	// User routes (protected)
	users := v1.Group("/users")
	users.Use(authMiddleware.RequireAuth())
	{
		users.GET("/me", userHandler.GetCurrentUser)
		users.GET("/:id", userHandler.GetUserByID)
	}

	// Project routes (protected)
	projects := v1.Group("/projects")
	projects.Use(authMiddleware.RequireAuth())
	{
		projects.POST("", projectHandler.CreateProject)
		projects.GET("", projectHandler.ListProjects)
		projects.GET("/:id", projectHandler.GetProject)
		projects.PUT("/:id", projectHandler.UpdateProject)
		projects.DELETE("/:id", projectHandler.DeleteProject)
	}

	// Volume routes (protected)
	volumes := v1.Group("/volumes")
	volumes.Use(authMiddleware.RequireAuth())
	{
		volumes.POST("", volumeHandler.AddVolume)
		volumes.GET("", volumeHandler.GetVolumes)
		volumes.DELETE("/:id", volumeHandler.RemoveVolume)
	}

	// Container routes (protected)
	// Note: We support both nested and top-level container routes
	projects.POST("/:id/containers", containerHandler.CreateContainer)

	containers := v1.Group("/containers")
	containers.Use(authMiddleware.RequireAuth())
	{
		containers.GET("", containerHandler.ListContainers)
		containers.GET("/:id", containerHandler.GetContainer)
		containers.PUT("/:id", containerHandler.UpdateContainer)
		containers.DELETE("/:id", containerHandler.DeleteContainer)

		// Git settings (uses UpdateContainer handler)
		containers.PUT("/:id/git", containerHandler.UpdateContainer)

		// Resource limits (uses UpdateContainer handler)
		containers.PUT("/:id/resources", containerHandler.UpdateContainer)

		// Environment variables
		containers.GET("/:id/env-vars", containerHandler.ListEnvVars)
		containers.POST("/:id/env-vars", containerHandler.AddEnvVar)
		containers.PUT("/:id/env-vars/:key", containerHandler.UpdateEnvVar)
		containers.DELETE("/:id/env-vars/:key", containerHandler.DeleteEnvVar)

		// Networks
		containers.GET("/:id/networks", containerHandler.ListNetworks)
		containers.POST("/:id/networks", containerHandler.AddNetwork)
		containers.DELETE("/:id/networks/:port", containerHandler.DeleteNetwork)

		// Secrets
		containers.GET("/:id/secrets", containerHandler.ListSecrets)
		containers.POST("/:id/secrets", containerHandler.AddSecret)
		containers.PUT("/:id/secrets/:key", containerHandler.UpdateSecret)
		containers.DELETE("/:id/secrets/:key", containerHandler.DeleteSecret)

		// Mounts
		containers.POST("/:id/mounts", containerHandler.AddMount)
		containers.DELETE("/:id/mounts/:volume_id", containerHandler.DeleteMount)
	}

	// Template routes (protected)
	templates := v1.Group("/templates")
	templates.Use(authMiddleware.RequireAuth())
	{
		templates.GET("", templateHandler.GetTemplates)
		templates.GET("/:id", templateHandler.GetTemplate)
	}

	return &TestServer{
		Router: router,
		DB:     testDB,
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

// MakeAuthRequest는 Bearer 토큰을 사용한 인증된 요청을 만듭니다
func (ts *TestServer) MakeAuthRequest(method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	// Bearer 토큰 헤더 추가
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

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

	// Activate user for testing (bypass email verification)
	// In production, users would verify email, but for E2E tests we activate directly
	_, err := ts.DB.DB.Exec("UPDATE USERS SET status = 'active' WHERE user_id = ?", userID)
	if err != nil {
		t.Fatalf("Failed to activate user: %v", err)
	}

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
