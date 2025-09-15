//go:build wireinject
// +build wireinject

package main

import (
	"database/sql"

	"github.com/google/wire"
	"github.com/swm-launchpad/web-console-backend/internal/auth"
	"github.com/swm-launchpad/web-console-backend/internal/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/auth/password"
	userssqlc "github.com/swm-launchpad/web-console-backend/internal/users/infrastructure/persistence/sqlc"
	"github.com/swm-launchpad/web-console-backend/internal/shared/config"
	"github.com/swm-launchpad/web-console-backend/internal/shared/db"
	"github.com/swm-launchpad/web-console-backend/internal/shared/middleware"
	"github.com/swm-launchpad/web-console-backend/internal/users/application/usecase"
	userHTTP "github.com/swm-launchpad/web-console-backend/internal/users/interfaces/http"
	"github.com/swm-launchpad/web-console-backend/internal/users/infrastructure/persistence"
)

func InitializeApp() (*App, error) {
	wire.Build(
		// Config
		config.Load,
		provideDatabase,
		wire.Bind(new(userssqlc.DBTX), new(*sql.DB)),

		// Auth infrastructure
		provideJWTService,
		password.NewService,
		auth.NewAuthService,

		// User domain
		persistence.NewUserRepository,

		// User use cases
		usecase.NewRegisterUserUseCase,
		usecase.NewLoginUserUseCase,
		usecase.NewGetUserUseCase,

		// HTTP handlers
		userHTTP.NewAuthHandler,
		userHTTP.NewUserHandler,

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

func provideJWTService(cfg *config.Config) *jwt.Service {
	return jwt.NewService(cfg.JWT.Secret)
}
