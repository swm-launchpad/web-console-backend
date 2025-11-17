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
	statusApp "github.com/swm-launchpad/web-console-backend/internal/status/application"
	statusService "github.com/swm-launchpad/web-console-backend/internal/status/domain/service"
	statusHTTP "github.com/swm-launchpad/web-console-backend/internal/status/handler"
	statusInfra "github.com/swm-launchpad/web-console-backend/internal/status/infrastructure"
	statusCron "github.com/swm-launchpad/web-console-backend/internal/status/infrastructure/cron"
	statusHealth "github.com/swm-launchpad/web-console-backend/internal/status/infrastructure/health"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
	userHTTP "github.com/swm-launchpad/web-console-backend/internal/user/handler"
	"github.com/swm-launchpad/web-console-backend/internal/user/infrastructure"
	userssqlc "github.com/swm-launchpad/web-console-backend/internal/user/infrastructure/sqlc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"time"
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

// provideJWTSecret extracts JWT secret from config
func provideJWTSecret(cfg *config.Config) string {
	return cfg.JWT.Secret
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
	registryURL := os.Getenv("REGISTRY_URL")
	if registryURL == "" {
		log.Fatal(nil, "REGISTRY_URL environment variable is required")
	}

	return projectInfra.NewContainerClient(
		getContainersForDeploymentUseCase,
		getContainersForBuildUseCase,
		getContainersForBuildAndDeployUseCase,
		registryURL,
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

// provideTektonNodePortClient creates a Tekton NodePort client from environment variables
func provideTektonNodePortClient(log logger.Logger) (containerDomainInfra.TektonNodePortClient, error) {
	return containerInfra.NewTektonNodePortClient(log)
}

// provideTektonCleanupClient creates a Tekton cleanup client from environment variables
func provideTektonCleanupClient(log logger.Logger) (projectDomainInfra.TektonCleanupClient, error) {
	return projectInfra.NewTektonCleanupClient(log)
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
		deployNamespace = "deploy-pipeline"
	}

	applicationNamespace := os.Getenv("KUBE_APPLICATION_NAMESPACE")
	if applicationNamespace == "" {
		applicationNamespace = "application"
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
		applicationNamespace,
		projectServiceName,
		log,
	)
}

// provideProjectQueries creates project sqlc queries
func provideProjectQueries(db projectSqlc.DBTX) *projectSqlc.Queries {
	return projectSqlc.New(db)
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
	listBranchesUseCase *application.ListBranchesUseCase,
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
		listBranchesUseCase,
		startInstallationUseCase,
		installationCallbackUseCase,
		cfg.Frontend.URL,
		log,
	)
}

// provideLokiClient creates a Loki client from config for container domain
func provideLokiClient(cfg *config.Config, log logger.Logger) containerDomainInfra.LokiClient {
	return containerInfra.NewLokiClient(cfg, log)
}

// provideProjectLokiClient creates a Loki client from config for project domain
func provideProjectLokiClient(cfg *config.Config, kubeClient projectDomainInfra.KubeClient, log logger.Logger) projectDomainInfra.LokiClient {
	return projectInfra.NewLokiClient(cfg, kubeClient, log)
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

// provideProjectLogHandler creates a project log handler with all dependencies
func provideProjectLogHandler(
	createTokenUC *projectApp.CreateProjectLogTokenUseCase,
	streamLogsUC *projectApp.StreamProjectLogsUseCase,
	historyUC *projectApp.GetProjectLogHistoryUseCase,
	projectRepo projectDomainRepo.ProjectRepository,
	permissionService projectService.PermissionService,
	jwtSecret string,
	log logger.Logger,
) *projectHTTP.ProjectLogHandler {
	return projectHTTP.NewProjectLogHandler(
		createTokenUC,
		streamLogsUC,
		historyUC,
		projectRepo,
		permissionService,
		jwtSecret,
		log,
	)
}

// provideKubernetesClientset creates a Kubernetes clientset
func provideKubernetesClientset(log logger.Logger) (*kubernetes.Clientset, error) {
	// Read configuration from environment variables
	apiServer := os.Getenv("KUBE_API_SERVER")
	if apiServer == "" {
		log.Warn(nil, "KUBE_API_SERVER not set, Kubernetes health check will be disabled")
		return nil, nil // Return nil if not configured
	}

	token := os.Getenv("KUBE_SERVICE_ACCOUNT_TOKEN")
	if token == "" {
		log.Warn(nil, "KUBE_SERVICE_ACCOUNT_TOKEN not set, Kubernetes health check will be disabled")
		return nil, nil
	}

	caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
	if caCertPath == "" {
		log.Warn(nil, "KUBE_CA_CERT_PATH not set, Kubernetes health check will be disabled")
		return nil, nil
	}

	// Verify CA cert file exists
	if _, err := os.Stat(caCertPath); err != nil {
		log.Warn(nil, "CA certificate file not found, Kubernetes health check will be disabled")
		return nil, nil
	}

	// Create REST config
	config := &rest.Config{
		Host:        apiServer,
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			CAFile: caCertPath,
		},
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(nil, "Failed to create Kubernetes clientset")
		return nil, nil // Return nil instead of error to allow app to start without K8s
	}

	return clientset, nil
}

// provideHealthCheckers creates all health checkers
func provideHealthCheckers(
	database *sql.DB,
	kubeClientset *kubernetes.Clientset,
) []statusService.HealthChecker {
	// Get configuration from environment
	webConsoleURL := os.Getenv("WEB_CONSOLE_HEALTH_CHECK_URL")
	if webConsoleURL == "" {
		webConsoleURL = "http://localhost:5173"
	}

	lokiURL := os.Getenv("LOKI_URL")
	if lokiURL == "" {
		lokiURL = "http://loki:3100"
	}

	registryURL := os.Getenv("REGISTRY_URL")
	if registryURL == "" {
		registryURL = "http://registry:5000"
	}

	tektonNamespace := os.Getenv("TEKTON_NAMESPACE")
	if tektonNamespace == "" {
		tektonNamespace = "tekton-pipelines"
	}

	nfsNamespace := os.Getenv("NFS_NAMESPACE")
	if nfsNamespace == "" {
		nfsNamespace = "nfs-provisioner"
	}

	nfsPVCName := os.Getenv("NFS_PVC_NAME")
	if nfsPVCName == "" {
		nfsPVCName = "nfs-pvc"
	}

	ingressNamespace := os.Getenv("INGRESS_NAMESPACE")
	if ingressNamespace == "" {
		ingressNamespace = "ingress-nginx"
	}

	timeout := 10 * time.Second

	checkers := []statusService.HealthChecker{
		statusHealth.NewAPIServerChecker(),
		statusHealth.NewMySQLChecker(database),
		statusHealth.NewWebConsoleChecker(webConsoleURL, timeout),
		statusHealth.NewLokiChecker(lokiURL, timeout),
		statusHealth.NewRegistryChecker(registryURL, timeout),
	}

	// Add Kubernetes-dependent checkers only if clientset is available
	if kubeClientset != nil {
		checkers = append(checkers,
			statusHealth.NewKubernetesChecker(kubeClientset),
			statusHealth.NewTektonChecker(kubeClientset, tektonNamespace),
			statusHealth.NewNFSChecker(kubeClientset, nfsNamespace, nfsPVCName),
			statusHealth.NewIngressChecker(kubeClientset, ingressNamespace),
		)
	}

	return checkers
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
		provideJWTSecret,
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
		application.NewListBranchesUseCase,
		application.NewStartInstallationUseCase,
		application.NewInstallationCallbackUseCase,

		// Project infrastructure
		projectRepo.NewProjectRepository,
		projectRepo.NewVolumeRepository,
		projectRepo.NewDeploymentRepository,
		projectRepo.NewBuildHistoryRepository,
		provideProjectQueries,
		provideTektonDeployClient,
		provideKubeDeployClient,
		provideContainerClient,
		provideTektonBuildClient,
		provideKubeBuildClient,
		provideTektonCleanupClient,
		provideProjectLokiClient,
		projectInfra.NewContainerUpdateAdapter,
		wire.Bind(new(projectDomainInfra.ContainerUpdater), new(*projectInfra.ContainerUpdateAdapter)),
		projectInfra.NewContainerSlugProvider,

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
		projectApp.NewDeployProjectUseCase,
		projectApp.NewGetProjectStatusUseCase,
		projectApp.NewRefreshProjectStatusUseCase,
		projectApp.NewCheckProjectPodStatusUseCase,
		projectApp.NewCreateProjectLogTokenUseCase,
		projectApp.NewStreamProjectLogsUseCase,
		projectApp.NewGetProjectLogHistoryUseCase,
		projectApp.NewAddVolumeUseCase,
		projectApp.NewGetVolumesUseCase,
		projectApp.NewRemoveVolumeUseCase,
		projectApp.NewCheckProjectNameUseCase,
		projectApp.NewCheckVolumeNameUseCase,

		// Container infrastructure
		containerInfra.NewContainerRepository,
		containerInfra.NewTemplateRepository,
		provideLokiClient,
		provideTektonNodePortClient,

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
		containerApp.NewUpdateNetworkUseCase,
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
		containerApp.NewGetContainerSlugsByProjectIDUseCase,
		containerApp.NewCheckFQDNUseCase,
		containerApp.NewCheckContainerNameUseCase,
		containerApp.NewCreateNodePortUseCase,
		containerApp.NewGetNodePortUseCase,

		// Status infrastructure
		provideKubernetesClientset,
		provideHealthCheckers,
		statusInfra.NewStatusRepository,
		statusInfra.NewIncidentRepository,

		// Status use cases
		statusApp.NewGetCurrentStatusUseCase,
		statusApp.NewGetStatusHistoryUseCase,
		statusApp.NewGetUptimeStatsUseCase,
		statusApp.NewGetDailyUptimeUseCase,
		statusApp.NewGetAllServiceHistoryUseCase,
		statusApp.NewPerformHealthChecksUseCase,
		statusApp.NewCalculateDailyUptimeUseCase,
		statusApp.NewCleanupOldChecksUseCase,

		// Status cron
		statusCron.NewStatusCron,

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
		provideProjectLogHandler,
		containerHTTP.NewContainerHandler,
		containerHTTP.NewTemplateHandler,
		provideBuildLogHandler,
		statusHTTP.NewStatusHandler,

		// Middleware
		middleware.NewAuthMiddleware,

		// Router and App
		NewRouter,
		NewApp,
	)
	return &App{}, nil
}
