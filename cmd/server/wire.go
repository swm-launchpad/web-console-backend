//go:build wireinject
// +build wireinject

package main

import (
	"database/sql"

	"github.com/google/wire"
	"github.com/swm-launchpad/web-console-backend/internal/shared/config"
	"github.com/swm-launchpad/web-console-backend/internal/shared/db"
)

func InitializeApp() (*App, error) {
	wire.Build(
		config.Load,
		provideDatabase,
		NewRouter,
		NewApp,
	)
	return nil, nil
}

func provideDatabase(cfg *config.Config) (*sql.DB, error) {
	return db.NewConnection(&cfg.Database)
}
