package repository

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
)

// StatusRepository defines the interface for status check persistence
type StatusRepository interface {
	// CreateStatusCheck stores a new status check result
	CreateStatusCheck(ctx context.Context, check *model.StatusCheck) error

	// GetLatestStatusCheck retrieves the most recent check for a service
	GetLatestStatusCheck(ctx context.Context, serviceName value.ServiceName) (*model.StatusCheck, error)

	// GetLatestStatusChecks retrieves the most recent checks for all services
	GetLatestStatusChecks(ctx context.Context) ([]*model.StatusCheck, error)

	// GetStatusChecksByPeriod retrieves all checks for a service within a time range
	GetStatusChecksByPeriod(ctx context.Context, serviceName value.ServiceName, start, end time.Time) ([]*model.StatusCheck, error)

	// GetDailyUptimeData retrieves aggregated daily uptime data
	GetDailyUptimeData(ctx context.Context, serviceName value.ServiceName, days int) ([]*model.DailyUptimeData, error)

	// UpsertDailyUptime inserts or updates daily uptime statistics
	UpsertDailyUptime(ctx context.Context, serviceName value.ServiceName, date time.Time, data *model.DailyUptimeData) error

	// DeleteOldStatusChecks removes status checks older than the specified date
	DeleteOldStatusChecks(ctx context.Context, olderThan time.Time) error

	// GetUptimeStats calculates uptime statistics for a service over a period
	GetUptimeStats(ctx context.Context, serviceName value.ServiceName, hours int) (*model.UptimeStats, error)
}
