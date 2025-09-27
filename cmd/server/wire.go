//go:build wireinject
// +build wireinject

package main

import (
	"database/sql"

	"github.com/google/wire"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/password"
	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/middleware"
	projectApp "github.com/swm-launchpad/web-console-backend/internal/project/application"
	projectService "github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	projectHTTP "github.com/swm-launchpad/web-console-backend/internal/project/handler"
	projectInfra "github.com/swm-launchpad/web-console-backend/internal/project/infrastructure"
	projectSqlc "github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/sqlc"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
	userHTTP "github.com/swm-launchpad/web-console-backend/internal/user/handler"
	"github.com/swm-launchpad/web-console-backend/internal/user/infrastructure"
	userssqlc "github.com/swm-launchpad/web-console-backend/internal/user/infrastructure/sqlc"
)

func InitializeApp() (*App, error) {
	wire.Build(
		// Config
		config.Load,
		provideDatabase,
		provideTxManager,
		wire.Bind(new(userssqlc.DBTX), new(*sql.DB)),
		wire.Bind(new(projectSqlc.DBTX), new(*sql.DB)),

		// Auth infrastructure
		provideJWTUtil,
		password.NewPasswordUtil,

		// User infrastructure
		infrastructure.NewUserRepository,

		// User domain services
		service.NewUserService,
		service.NewAuthService,

		// User use cases
		application.NewRegisterUserUseCase,
		application.NewLoginUserUseCase,
		application.NewGetUserUseCase,

		// Project infrastructure
		projectInfra.NewProjectRepository,
		projectInfra.NewVolumeRepository,

		// Project domain services
		projectService.NewSlugService,
		projectService.NewProjectService,
		projectService.NewVolumeService,
		projectService.NewPermissionService,

		// Project use cases
		projectApp.NewCreateProjectUseCase,
		projectApp.NewGetProjectUseCase,
		projectApp.NewUpdateProjectUseCase,
		projectApp.NewDeleteProjectUseCase,
		projectApp.NewListProjectsUseCase,
		projectApp.NewAddVolumeUseCase,
		projectApp.NewGetVolumesUseCase,
		projectApp.NewUpdateVolumeUseCase,
		projectApp.NewRemoveVolumeUseCase,

		// HTTP handlers
		userHTTP.NewAuthHandler,
		userHTTP.NewUserHandler,
		projectHTTP.NewProjectHandler,
		projectHTTP.NewVolumeHandler,

		// Middleware
		middleware.NewAuthMiddleware,

		// Router and App
		NewRouter,
		NewApp,
	)
	return nil, nil
}

func provideDatabase(cfg *config.Config) (*sql.DB, error) {
	return db.NewConnection(&cfg.Database)
}

func provideTxManager(database *sql.DB) db.TxManager {
	return db.NewTxManager(database)
}

func provideJWTUtil(cfg *config.Config) *jwt.JWTUtil {
	return jwt.NewJWTUtil(cfg.JWT.Secret)
}
