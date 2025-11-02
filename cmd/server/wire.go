//go:build wireinject
// +build wireinject

package main

import (
	"database/sql"
	"os"

	"github.com/google/wire"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/password"
	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/email"
	"github.com/swm-launchpad/web-console-backend/internal/common/github"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/middleware"
	"github.com/swm-launchpad/web-console-backend/internal/common/settings"
	containerApp "github.com/swm-launchpad/web-console-backend/internal/container/application"
	containerBuild "github.com/swm-launchpad/web-console-backend/internal/container/application/build"
	containerCombined "github.com/swm-launchpad/web-console-backend/internal/container/application/combined"
	containerDeployment "github.com/swm-launchpad/web-console-backend/internal/container/application/deployment"
	containerDomainInfra "github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure"
	containerService "github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	containerHTTP "github.com/swm-launchpad/web-console-backend/internal/container/handler"
	containerInfra "github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
	containerSqlc "github.com/swm-launchpad/web-console-backend/internal/container/infrastructure/sqlc"
	projectApp "github.com/swm-launchpad/web-console-backend/internal/project/application"
	projectDomainInfra "github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	projectDomainRepo "github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	projectService "github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	projectBuildService "github.com/swm-launchpad/web-console-backend/internal/project/domain/service/build"
	projectDeployService "github.com/swm-launchpad/web-console-backend/internal/project/domain/service/deploy"
	projectHTTP "github.com/swm-launchpad/web-console-backend/internal/project/handler"
	projectInfra "github.com/swm-launchpad/web-console-backend/internal/project/infrastructure"
	projectRepo "github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/repository"
	projectSqlc "github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/repository/sqlc"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
	userHTTP "github.com/swm-launchpad/web-console-backend/internal/user/handler"
	"github.com/swm-launchpad/web-console-backend/internal/user/infrastructure"
	userssqlc "github.com/swm-launchpad/web-console-backend/internal/user/infrastructure/sqlc"
)

// provideDatabase creates a database connection from config
func provideDatabase(cfg *config.Config) (*sql.DB, error) {
	return db.NewConnection(&cfg.Database)
}

// provideTxManager creates a transaction manager
func provideTxManager(database *sql.DB) db.TxManager {
	return db.NewTxManager(database)
}

// provideJWTUtil creates a JWT utility from config
func provideJWTUtil(cfg *config.Config) *jwt.JWTUtil {
	return jwt.NewJWTUtil(cfg.JWT.Secret)
}

// provideLogger creates a logger from config
func provideLogger(cfg *config.Config) (logger.Logger, error) {
	loggerCfg := logger.Config{
		Level:    cfg.Log.Level,
		Format:   cfg.Log.Format,
		FilePath: cfg.Log.FilePath,
	}
	return logger.New(loggerCfg)
}

// provideLoggingMiddleware creates a logging middleware
func provideLoggingMiddleware(log logger.Logger) *logger.LoggingMiddleware {
	return logger.NewLoggingMiddleware(log)
}

// provideEmailService creates an email service from config
func provideEmailService(cfg *config.Config, log logger.Logger) email.Service {
	// For development, use default values if not configured
	host := cfg.Email.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.Email.Port
	if port == 0 {
		port = 587
	}
	from := cfg.Email.From
	if from == "" {
		from = "noreply@localhost"
	}
	frontendURL := cfg.Frontend.URL
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	return email.NewService(
		host,
		port,
		cfg.Email.Username,
		cfg.Email.Password,
		from,
		frontendURL,
		log,
	)
}

// provideTektonDeployClient creates a Tekton client from environment variables
func provideTektonDeployClient(log logger.Logger) (projectDomainInfra.TektonClient, error) {
	return projectInfra.NewTektonDeployClient(log)
}

// provideKubeDeployClient creates a Kubernetes client from environment variables
func provideKubeDeployClient(log logger.Logger) (projectDomainInfra.KubeClient, error) {
	return projectInfra.NewKubeDeployClient(log)
}

// provideContainerClient creates a container client
func provideContainerClient(
	getContainersForDeploymentUseCase *containerDeployment.GetContainersForDeploymentUseCase,
	getContainersForBuildUseCase *containerBuild.GetContainersForBuildUseCase,
	getContainersForBuildAndDeployUseCase *containerCombined.GetContainersForBuildAndDeployUseCase,
	log logger.Logger,
) projectDomainInfra.ContainerClient {
	return projectInfra.NewContainerClient(
		getContainersForDeploymentUseCase,
		getContainersForBuildUseCase,
		getContainersForBuildAndDeployUseCase,
		log,
	)
}

// provideKubeBuildClient creates a Kubernetes build client from environment variables
func provideKubeBuildClient(log logger.Logger) (projectDomainInfra.KubeBuildClient, error) {
	return projectInfra.NewKubeBuildClient(log)
}

// provideTektonBuildClient creates a Tekton build client from environment variables
func provideTektonBuildClient(log logger.Logger) (projectDomainInfra.TektonBuildClient, error) {
	return projectInfra.NewTektonBuildClient(log)
}

// provideDeployNamespace provides the deployment namespace from environment
func provideDeployNamespace() string {
	// This is used by DeployService to know which namespace to deploy to
	// Read from same env var as KubeClient
	deployNamespace := os.Getenv("KUBE_DEPLOY_NAMESPACE")
	if deployNamespace == "" {
		return "default"
	}
	return deployNamespace
}

// provideDeployService creates a DeployService with all dependencies
func provideDeployService(
	txManager db.TxManager,
	projectRepository projectDomainRepo.ProjectRepository,
	deploymentRepo projectDomainRepo.DeploymentRepository,
	buildHistoryRepo projectDomainRepo.BuildHistoryRepository,
	volumeRepo projectDomainRepo.VolumeRepository,
	containerClient projectDomainInfra.ContainerClient,
	tektonClient projectDomainInfra.TektonClient,
	kubeClient projectDomainInfra.KubeClient,
	kubeBuildClient projectDomainInfra.KubeBuildClient,
	buildOrchestrator projectBuildService.Orchestrator,
	buildPostProcessor projectBuildService.PostProcessor,
	log logger.Logger,
) projectDeployService.Deployer {
	deployNamespace := os.Getenv("KUBE_DEPLOY_NAMESPACE")
	if deployNamespace == "" {
		deployNamespace = "default"
	}
	// projectServiceName is not used in the actual implementation
	projectServiceName := ""

	return projectDeployService.NewDeployer(
		txManager,
		projectRepository,
		deploymentRepo,
		buildHistoryRepo,
		volumeRepo,
		containerClient,
		tektonClient,
		kubeClient,
		kubeBuildClient,
		buildOrchestrator,
		buildPostProcessor,
		deployNamespace,
		projectServiceName,
		log,
	)
}

// provideGitHubClient creates a GitHub client from config
// Returns nil if GitHub App credentials are not configured
func provideGitHubClient(cfg *config.Config) (*github.Client, error) {
	// GitHub client is optional - return nil if not configured
	if cfg.GitHubApp.AppID == "" || cfg.GitHubApp.PrivateKeyPath == "" {
		return nil, nil
	}
	return github.NewClient(cfg.GitHubApp.AppID, cfg.GitHubApp.PrivateKeyPath)
}

// provideGitHubHandler creates a GitHub handler with frontend URL
func provideGitHubHandler(
	connectUseCase *application.ConnectGitHubUseCase,
	disconnectUseCase *application.DisconnectGitHubUseCase,
	getInstallationUseCase *application.GetGitHubInstallationUseCase,
	generateTokenUseCase *application.GenerateInstallationTokenUseCase,
	listRepositoriesUseCase *application.ListRepositoriesUseCase,
	startInstallationUseCase *application.StartInstallationUseCase,
	installationCallbackUseCase *application.InstallationCallbackUseCase,
	cfg *config.Config,
	log logger.Logger,
) *userHTTP.GitHubHandler {
	return userHTTP.NewGitHubHandler(
		connectUseCase,
		disconnectUseCase,
		getInstallationUseCase,
		generateTokenUseCase,
		listRepositoriesUseCase,
		startInstallationUseCase,
		installationCallbackUseCase,
		cfg.Frontend.URL,
		log,
	)
}

// provideLokiClient creates a Loki client from config
func provideLokiClient(cfg *config.Config, log logger.Logger) containerDomainInfra.LokiClient {
	return containerInfra.NewLokiClient(cfg, log)
}

// provideBuildLogHandler creates a build log handler with all dependencies
func provideBuildLogHandler(
	createBuildLogTokenUC *containerApp.CreateBuildLogTokenUseCase,
	streamBuildLogsUC *containerApp.StreamBuildLogsUseCase,
	getBuildLogHistoryUC *containerApp.GetBuildLogHistoryUseCase,
	containerService containerService.ContainerService,
	jwtUtil *jwt.JWTUtil,
	log logger.Logger,
) *containerHTTP.BuildLogHandler {
	return containerHTTP.NewBuildLogHandler(
		createBuildLogTokenUC,
		streamBuildLogsUC,
		getBuildLogHistoryUC,
		containerService,
		jwtUtil,
		log,
	)
}

func InitializeApp() (*App, error) {
	wire.Build(
		// Config
		config.Load,
		provideDatabase,
		provideTxManager,
		wire.Bind(new(userssqlc.DBTX), new(*sql.DB)),
		wire.Bind(new(projectSqlc.DBTX), new(*sql.DB)),
		wire.Bind(new(containerSqlc.DBTX), new(*sql.DB)),

		// Logger
		provideLogger,
		provideLoggingMiddleware,

		// Auth infrastructure
		provideJWTUtil,
		password.NewPasswordUtil,

		// Email service
		provideEmailService,

		// GitHub client
		provideGitHubClient,

		// Settings service
		settings.NewSettingsRepository,
		settings.NewSettingsService,
		settings.NewSettingsHandler,

		// User infrastructure
		infrastructure.NewUserRepository,
		infrastructure.NewTokenRepository,
		infrastructure.NewGitHubInstallationRepository,
		infrastructure.NewOAuthStateRepository,

		// User domain services
		service.NewUserService,
		service.NewAuthService,
		service.NewTokenService,

		// User use cases
		application.NewRegisterUserUseCase,
		application.NewLoginUserUseCase,
		application.NewGetUserUseCase,
		application.NewUpdateUserUseCase,
		application.NewChangePasswordUseCase,
		application.NewVerifyEmailUseCase,
		application.NewResendVerificationEmailUseCase,
		application.NewRequestPasswordResetUseCase,
		application.NewResetPasswordUseCase,
		application.NewConnectGitHubUseCase,
		application.NewDisconnectGitHubUseCase,
		application.NewGetGitHubInstallationUseCase,
		application.NewGenerateInstallationTokenUseCase,
		application.NewListRepositoriesUseCase,
		application.NewStartInstallationUseCase,
		application.NewInstallationCallbackUseCase,

		// Project infrastructure
		projectRepo.NewProjectRepository,
		projectRepo.NewVolumeRepository,
		projectRepo.NewDeploymentRepository,
		projectRepo.NewBuildHistoryRepository,
		provideTektonDeployClient,
		provideKubeDeployClient,
		provideContainerClient,
		provideTektonBuildClient,
		provideKubeBuildClient,
		projectInfra.NewContainerUpdateAdapter,
		wire.Bind(new(projectDomainInfra.ContainerUpdater), new(*projectInfra.ContainerUpdateAdapter)),

		// Project domain services
		projectService.NewSlugService,
		projectService.NewVolumeSlugService,
		projectService.NewValidationService,
		projectService.NewProjectService,
		projectService.NewVolumeService,
		projectService.NewPermissionService,
		projectBuildService.NewBuilder,
		projectBuildService.NewOrchestrator,
		projectBuildService.NewPostProcessor,
		provideDeployService,

		// Project use cases
		projectApp.NewCreateProjectUseCase,
		projectApp.NewGetProjectUseCase,
		projectApp.NewGetProjectBySlugUseCase,
		projectApp.NewUpdateProjectUseCase,
		projectApp.NewDeleteProjectUseCase,
		projectApp.NewListProjectsUseCase,
		projectApp.NewAddVolumeUseCase,
		projectApp.NewGetVolumesUseCase,
		projectApp.NewRemoveVolumeUseCase,
		projectApp.NewDeployProjectUseCase,
		projectApp.NewGetProjectStatusUseCase,
		projectApp.NewRefreshProjectStatusUseCase,

		// Container infrastructure
		containerInfra.NewContainerRepository,
		containerInfra.NewTemplateRepository,
		provideLokiClient,

		// Container domain services
		containerService.NewSlugService,
		containerService.NewContainerService,
		containerService.NewPermissionService,
		containerService.NewResourceValidationService,
		containerService.NewBuildChangeDetector,

		// Container use cases
		containerApp.NewCreateContainerUseCase,
		containerApp.NewGetContainerUseCase,
		containerApp.NewUpdateContainerUseCase,
		containerApp.NewDeleteContainerUseCase,
		containerApp.NewListContainersUseCase,
		containerApp.NewAddEnvVarUseCase,
		containerApp.NewUpdateEnvVarUseCase,
		containerApp.NewDeleteEnvVarUseCase,
		containerApp.NewAddNetworkUseCase,
		containerApp.NewDeleteNetworkUseCase,
		containerApp.NewAddSecretUseCase,
		containerApp.NewUpdateSecretUseCase,
		containerApp.NewDeleteSecretUseCase,
		containerApp.NewAddBuildVarUseCase,
		containerApp.NewUpdateBuildVarUseCase,
		containerApp.NewDeleteBuildVarUseCase,
		containerApp.NewAddMountUseCase,
		containerApp.NewDeleteMountUseCase,
		containerApp.NewGetTemplatesUseCase,
		containerApp.NewGetTemplateUseCase,
		containerDeployment.NewGetContainersForDeploymentUseCase,
		containerBuild.NewGetContainersForBuildUseCase,
		containerCombined.NewGetContainersForBuildAndDeployUseCase,
		containerBuild.NewUpdateContainerAfterBuildUseCase,
		containerApp.NewCreateBuildLogTokenUseCase,
		containerApp.NewStreamBuildLogsUseCase,
		containerApp.NewGetBuildLogHistoryUseCase,

		// HTTP handlers
		userHTTP.NewAuthHandler,
		userHTTP.NewUserHandler,
		userHTTP.NewVerificationHandler,
		userHTTP.NewPasswordResetHandler,
		provideGitHubHandler,
		projectHTTP.NewProjectHandler,
		projectHTTP.NewVolumeHandler,
		projectHTTP.NewDeploymentHandler,
		projectHTTP.NewProjectStatusHandler,
		containerHTTP.NewContainerHandler,
		containerHTTP.NewTemplateHandler,
		provideBuildLogHandler,

		// Middleware
		middleware.NewAuthMiddleware,

		// Router and App
		NewRouter,
		NewApp,
	)
	return &App{}, nil
}
