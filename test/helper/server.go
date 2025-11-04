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
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/middleware"
	"github.com/swm-launchpad/web-console-backend/internal/common/settings"
	containerApp "github.com/swm-launchpad/web-console-backend/internal/container/application"
	containerService "github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	containerHTTP "github.com/swm-launchpad/web-console-backend/internal/container/handler"
	containerInfra "github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
	projectApp "github.com/swm-launchpad/web-console-backend/internal/project/application"
	projectService "github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	projectHTTP "github.com/swm-launchpad/web-console-backend/internal/project/handler"
	projectRepo "github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	userrepository "github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
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
	userRepo := infrastructure.NewUserRepository(testDB.DB, logger.NewForTest())
	tokenRepo := infrastructure.NewTokenRepository(testDB.DB, logger.NewForTest())
	installationRepo := infrastructure.NewGitHubInstallationRepository(testDB.DB, logger.NewForTest())
	jwtUtil := jwt.NewJWTUtil("test-secret")
	passwordUtil := password.NewPasswordUtil()
	txManager := db.NewTxManager(testDB.DB)

	// Email Service 초기화 (테스트용 Mock 사용)
	mockEmailService := new(email.MockService)
	// Mock email service to always succeed (ctx, email, username, token)
	mockEmailService.On("SendVerificationEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockEmailService.On("SendPasswordResetEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Logger 초기화 (테스트용)
	testLogger := logger.NewForTest()

	// User Service 초기화
	userService := service.NewUserService(userRepo, testLogger)
	authService := service.NewAuthService(userService, jwtUtil, passwordUtil, testLogger)
	tokenService := service.NewTokenService(tokenRepo, testLogger)

	// User UseCase 초기화
	registerUseCase := application.NewRegisterUserUseCase(authService, tokenService, mockEmailService, txManager, testLogger)
	loginUseCase := application.NewLoginUserUseCase(authService, testLogger)
	getUserUseCase := application.NewGetUserUseCase(userService, testLogger)
	updateUserUseCase := application.NewUpdateUserUseCase(userService, testLogger)
	changePasswordUseCase := application.NewChangePasswordUseCase(userService, authService, testLogger)

	// Settings service (for validation)
	settingsRepo := settings.NewSettingsRepository(testDB.DB)
	settingsSvc := settings.NewSettingsService(settingsRepo)

	// Project dependencies
	projectRepository := projectRepo.NewProjectRepository(testDB.DB, testLogger)
	volumeRepo := projectRepo.NewVolumeRepository(testDB.DB, testLogger)
	slugService := projectService.NewSlugService(projectRepository, testLogger)
	volumeSlugService := projectService.NewVolumeSlugService(volumeRepo, testLogger)
	validationService := projectService.NewValidationService(projectRepository, settingsSvc)
	projectSvc := projectService.NewProjectService(projectRepository, slugService, validationService, testLogger)
	volumeSvc := projectService.NewVolumeService(volumeRepo, projectRepository, volumeSlugService, testLogger)
	permissionSvc := projectService.NewPermissionService(projectRepository, volumeRepo, testLogger)

	// Project UseCases
	createProjectUseCase := projectApp.NewCreateProjectUseCase(projectSvc, txManager, testLogger)
	getProjectUseCase := projectApp.NewGetProjectUseCase(projectSvc, volumeSvc, testLogger)
	getProjectBySlugUseCase := projectApp.NewGetProjectBySlugUseCase(projectSvc, volumeSvc, testLogger)
	updateProjectUseCase := projectApp.NewUpdateProjectUseCase(projectSvc, txManager, testLogger)
	deleteProjectUseCase := projectApp.NewDeleteProjectUseCase(projectSvc, volumeSvc, txManager, testLogger)
	listProjectsUseCase := projectApp.NewListProjectsUseCase(projectSvc, testLogger)

	// Container dependencies
	containerRepo := containerInfra.NewContainerRepository(testDB.DB, testLogger)
	templateRepo := containerInfra.NewTemplateRepository(testDB.DB, testLogger)
	containerSlugService := containerService.NewSlugService(containerRepo, testLogger)
	containerSvc := containerService.NewContainerService(containerRepo, containerSlugService, testLogger)
	containerPermissionSvc := containerService.NewPermissionService(containerRepo, projectRepository, testLogger)
	resourceValidationSvc := containerService.NewResourceValidationService(containerRepo, projectRepository, testLogger)
	buildChangeDetector := containerService.NewBuildChangeDetector()

	// Container UseCases
	createContainerUseCase := containerApp.NewCreateContainerUseCase(containerSvc, containerRepo, templateRepo, containerPermissionSvc, resourceValidationSvc, volumeSvc, installationRepo, txManager, testLogger)
	getContainerUseCase := containerApp.NewGetContainerUseCase(containerRepo, containerPermissionSvc, testLogger)
	listContainersUseCase := containerApp.NewListContainersUseCase(containerRepo, containerPermissionSvc, testLogger)
	updateContainerUseCase := containerApp.NewUpdateContainerUseCase(containerRepo, containerPermissionSvc, resourceValidationSvc, buildChangeDetector, installationRepo, txManager, testLogger)
	deleteContainerUseCase := containerApp.NewDeleteContainerUseCase(containerRepo, containerPermissionSvc, volumeSvc, txManager, testLogger)
	addEnvVarUseCase := containerApp.NewAddEnvVarUseCase(containerRepo, containerPermissionSvc, txManager, testLogger)
	updateEnvVarUseCase := containerApp.NewUpdateEnvVarUseCase(containerRepo, containerPermissionSvc, txManager, testLogger)
	deleteEnvVarUseCase := containerApp.NewDeleteEnvVarUseCase(containerRepo, containerPermissionSvc, txManager, testLogger)
	addNetworkUseCase := containerApp.NewAddNetworkUseCase(containerRepo, containerPermissionSvc, txManager, testLogger)
	deleteNetworkUseCase := containerApp.NewDeleteNetworkUseCase(containerRepo, containerPermissionSvc, txManager, testLogger)
	addSecretUseCase := containerApp.NewAddSecretUseCase(containerRepo, containerPermissionSvc, txManager, testLogger)
	updateSecretUseCase := containerApp.NewUpdateSecretUseCase(containerRepo, containerPermissionSvc, txManager, testLogger)
	deleteSecretUseCase := containerApp.NewDeleteSecretUseCase(containerRepo, containerPermissionSvc, txManager, testLogger)
	addBuildVarUseCase := containerApp.NewAddBuildVarUseCase(containerRepo, containerPermissionSvc, txManager, testLogger)
	updateBuildVarUseCase := containerApp.NewUpdateBuildVarUseCase(containerRepo, containerPermissionSvc, txManager, testLogger)
	deleteBuildVarUseCase := containerApp.NewDeleteBuildVarUseCase(containerRepo, containerPermissionSvc, txManager, testLogger)
	addMountUseCase := containerApp.NewAddMountUseCase(containerRepo, containerPermissionSvc, volumeSvc, txManager, testLogger)
	deleteMountUseCase := containerApp.NewDeleteMountUseCase(containerRepo, containerPermissionSvc, txManager, testLogger)

	// Template UseCases
	getTemplatesUseCase := containerApp.NewGetTemplatesUseCase(templateRepo, testLogger)
	getTemplateUseCase := containerApp.NewGetTemplateUseCase(templateRepo, testLogger)

	// Handler 초기화
	authHandler := userhttp.NewAuthHandler(registerUseCase, loginUseCase, testLogger)
	userHandler := userhttp.NewUserHandler(getUserUseCase, updateUserUseCase, changePasswordUseCase, testLogger)
	projectHandler := projectHTTP.NewProjectHandler(
		createProjectUseCase,
		getProjectUseCase,
		getProjectBySlugUseCase,
		updateProjectUseCase,
		deleteProjectUseCase,
		listProjectsUseCase,
		permissionSvc,
		projectSvc,
		settingsSvc,
		testLogger,
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
		addBuildVarUseCase,
		updateBuildVarUseCase,
		deleteBuildVarUseCase,
		addMountUseCase,
		deleteMountUseCase,
		projectSvc,
		volumeSvc,
		containerSvc,
		containerPermissionSvc,
		testLogger,
	)
	templateHandler := containerHTTP.NewTemplateHandler(
		getTemplatesUseCase,
		getTemplateUseCase,
		testLogger,
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
		projects.GET("/:slug", projectHandler.GetProject)
		projects.PUT("/:slug", projectHandler.UpdateProject)
		projects.DELETE("/:slug", projectHandler.DeleteProject)

		// Container routes under project (RESTful)
		projects.POST("/:slug/containers", containerHandler.CreateContainer)
		projects.GET("/:slug/containers", containerHandler.ListContainers)
	}

	// Container routes (protected, slug-based)
	containers := v1.Group("/containers")
	containers.Use(authMiddleware.RequireAuth())
	{
		containers.GET("/:slug", containerHandler.GetContainer)
		containers.PUT("/:slug", containerHandler.UpdateContainer)
		containers.DELETE("/:slug", containerHandler.DeleteContainer)

		// Git settings
		containers.PUT("/:slug/git", containerHandler.UpdateContainer)

		// Resource limits
		containers.PUT("/:slug/resources", containerHandler.UpdateContainer)

		// Environment variables
		containers.POST("/:slug/env-vars", containerHandler.AddEnvVar)
		containers.PUT("/:slug/env-vars/:key", containerHandler.UpdateEnvVar)
		containers.DELETE("/:slug/env-vars/:key", containerHandler.DeleteEnvVar)

		// Networks
		containers.POST("/:slug/networks", containerHandler.AddNetwork)
		containers.DELETE("/:slug/networks/:port", containerHandler.DeleteNetwork)

		// Secrets
		containers.POST("/:slug/secrets", containerHandler.AddSecret)
		containers.PUT("/:slug/secrets/:key", containerHandler.UpdateSecret)
		containers.DELETE("/:slug/secrets/:key", containerHandler.DeleteSecret)

		// Volumes (container sub-resource)
		containers.GET("/:slug/volumes", containerHandler.ListVolumes)
		containers.POST("/:slug/volumes", containerHandler.AddVolume)
		containers.DELETE("/:slug/volumes/:volume_id", containerHandler.DeleteVolume)
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

// GetGitHubInstallationRepository returns the GitHub installation repository for testing
func GetGitHubInstallationRepository(testDB *TestDB) userrepository.GitHubInstallationRepository {
	return infrastructure.NewGitHubInstallationRepository(testDB.DB, logger.NewForTest())
}

// GetCreateContainerUseCase returns the create container use case for testing
func GetCreateContainerUseCase(ts *TestServer) *containerApp.CreateContainerUseCase {
	testLogger := logger.NewForTest()
	containerRepo := containerInfra.NewContainerRepository(ts.DB.DB, testLogger)
	templateRepo := containerInfra.NewTemplateRepository(ts.DB.DB, testLogger)
	containerSlugService := containerService.NewSlugService(containerRepo, testLogger)
	containerSvc := containerService.NewContainerService(containerRepo, containerSlugService, testLogger)
	projectRepository := projectRepo.NewProjectRepository(ts.DB.DB, testLogger)
	containerPermissionSvc := containerService.NewPermissionService(containerRepo, projectRepository, testLogger)
	resourceValidationSvc := containerService.NewResourceValidationService(containerRepo, projectRepository, testLogger)
	volumeRepo := projectRepo.NewVolumeRepository(ts.DB.DB, testLogger)
	volumeSlugService := projectService.NewVolumeSlugService(volumeRepo, testLogger)
	volumeSvc := projectService.NewVolumeService(volumeRepo, projectRepository, volumeSlugService, testLogger)
	installationRepo := infrastructure.NewGitHubInstallationRepository(ts.DB.DB, testLogger)
	txManager := db.NewTxManager(ts.DB.DB)
	return containerApp.NewCreateContainerUseCase(
		containerSvc,
		containerRepo,
		templateRepo,
		containerPermissionSvc,
		resourceValidationSvc,
		volumeSvc,
		installationRepo,
		txManager,
		testLogger,
	)
}

// GetUpdateContainerUseCase returns the update container use case for testing
func GetUpdateContainerUseCase(ts *TestServer) *containerApp.UpdateContainerUseCase {
	testLogger := logger.NewForTest()
	containerRepo := containerInfra.NewContainerRepository(ts.DB.DB, testLogger)
	projectRepository := projectRepo.NewProjectRepository(ts.DB.DB, testLogger)
	containerPermissionSvc := containerService.NewPermissionService(containerRepo, projectRepository, testLogger)
	resourceValidationSvc := containerService.NewResourceValidationService(containerRepo, projectRepository, testLogger)
	buildChangeDetector := containerService.NewBuildChangeDetector()
	installationRepo := infrastructure.NewGitHubInstallationRepository(ts.DB.DB, testLogger)
	txManager := db.NewTxManager(ts.DB.DB)

	return containerApp.NewUpdateContainerUseCase(
		containerRepo,
		containerPermissionSvc,
		resourceValidationSvc,
		buildChangeDetector,
		installationRepo,
		txManager,
		testLogger,
	)
}

// GetProjectDiskUsage returns the total disk usage for a project
func (ts *TestServer) GetProjectDiskUsage(t *testing.T, projectID uint) uint32 {
	t.Helper()

	var totalDisk uint32
	query := `
		SELECT COALESCE(SUM(capacity), 0) as total_disk
		FROM VOLUMES
		WHERE project_id = ?
	`
	err := ts.DB.DB.QueryRow(query, projectID).Scan(&totalDisk)
	if err != nil {
		t.Fatalf("Failed to get project disk usage: %v", err)
	}

	return totalDisk
}

// GetContainerStatus returns whether the container exists and is deleted
func (ts *TestServer) GetContainerStatus(t *testing.T, containerID uint) (exists bool, isDeleted bool) {
	t.Helper()

	var deletedAt *string
	query := `
		SELECT deleted_at
		FROM CONTAINERS
		WHERE container_id = ?
	`
	err := ts.DB.DB.QueryRow(query, containerID).Scan(&deletedAt)
	if err != nil {
		// Container does not exist
		return false, false
	}

	// Container exists
	exists = true
	isDeleted = (deletedAt != nil)
	return exists, isDeleted
}

// VolumeExists checks if a volume exists in the database
func (ts *TestServer) VolumeExists(t *testing.T, volumeID uint) bool {
	t.Helper()

	var count int
	query := `
		SELECT COUNT(*)
		FROM VOLUMES
		WHERE volume_id = ?
	`
	err := ts.DB.DB.QueryRow(query, volumeID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check if volume exists: %v", err)
	}

	return count > 0
}
