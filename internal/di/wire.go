//go:build wireinject
// +build wireinject

package di

import (
	"database/sql"

	"github.com/google/wire"
	"github.com/swm-launchpad/web-console-backend/internal/interfaces/http/router"
	"github.com/swm-launchpad/web-console-backend/internal/shared/config"
	"github.com/swm-launchpad/web-console-backend/internal/shared/db"
)

func InitializeApp() (*App, error) {
	wire.Build(
		config.Load,
		provideDatabase,
		router.New,
		NewApp,
	)
	return nil, nil
}

func provideDatabase(cfg *config.Config) (*sql.DB, error) {
	return db.NewConnection(&cfg.Database)
}
