package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/repository"
	"github.com/swm-launchpad/web-console-backend/internal/status/infrastructure/sqlc"
)

// StatusRepositoryImpl implements the StatusRepository interface using SQLC
type StatusRepositoryImpl struct {
	db      *sql.DB
	queries *sqlc.Queries
}

// NewStatusRepository creates a new StatusRepositoryImpl
func NewStatusRepository(db *sql.DB) repository.StatusRepository {
	return &StatusRepositoryImpl{
		db:      db,
		queries: sqlc.New(db),
	}
}

// CreateStatusCheck stores a new status check result
func (r *StatusRepositoryImpl) CreateStatusCheck(ctx context.Context, check *model.StatusCheck) error {
	metadata, err := json.Marshal(check.Metadata)
	if err != nil {
		return err
	}

	var responseTimeMs sql.NullInt32
	if check.ResponseTimeMs != nil {
		responseTimeMs = sql.NullInt32{Int32: int32(*check.ResponseTimeMs), Valid: true}
	}

	var errorMsg sql.NullString
	if check.ErrorMessage != nil {
		errorMsg = sql.NullString{String: *check.ErrorMessage, Valid: true}
	}

	_, err = r.queries.CreateStatusCheck(ctx, sqlc.CreateStatusCheckParams{
		ServiceName:     check.ServiceName.String(),
		ServiceCategory: sqlc.ServiceStatusChecksServiceCategory(check.ServiceCategory.String()),
		Status:          sqlc.ServiceStatusChecksStatus(check.Status.String()),
		ResponseTimeMs:  responseTimeMs,
		ErrorMessage:    errorMsg,
		Metadata:        metadata,
	})

	return err
}

// GetLatestStatusCheck retrieves the most recent check for a service
func (r *StatusRepositoryImpl) GetLatestStatusCheck(ctx context.Context, serviceName value.ServiceName) (*model.StatusCheck, error) {
	row, err := r.queries.GetLatestStatusCheck(ctx, serviceName.String())
	if err != nil {
		return nil, err
	}

	return r.rowToStatusCheck(row)
}

// GetLatestStatusChecks retrieves the most recent checks for all services
func (r *StatusRepositoryImpl) GetLatestStatusChecks(ctx context.Context) ([]*model.StatusCheck, error) {
	rows, err := r.queries.GetLatestStatusChecks(ctx)
	if err != nil {
		return nil, err
	}

	checks := make([]*model.StatusCheck, 0, len(rows))
	for _, row := range rows {
		check, err := r.rowToStatusCheck(sqlc.ServiceStatusCheck{
			CheckID:         row.CheckID,
			ServiceName:     row.ServiceName,
			ServiceCategory: row.ServiceCategory,
			Status:          row.Status,
			ResponseTimeMs:  row.ResponseTimeMs,
			ErrorMessage:    row.ErrorMessage,
			CheckedAt:       row.CheckedAt,
			Metadata:        row.Metadata,
		})
		if err != nil {
			continue
		}
		checks = append(checks, check)
	}

	return checks, nil
}

// GetStatusChecksByPeriod retrieves all checks for a service within a time range
func (r *StatusRepositoryImpl) GetStatusChecksByPeriod(ctx context.Context, serviceName value.ServiceName, start, end time.Time) ([]*model.StatusCheck, error) {
	rows, err := r.queries.GetStatusChecksByPeriod(ctx, sqlc.GetStatusChecksByPeriodParams{
		ServiceName:   serviceName.String(),
		FromCheckedAt: start,
		ToCheckedAt:   end,
	})
	if err != nil {
		return nil, err
	}

	checks := make([]*model.StatusCheck, 0, len(rows))
	for _, row := range rows {
		check, err := r.rowToStatusCheck(row)
		if err != nil {
			continue
		}
		checks = append(checks, check)
	}

	return checks, nil
}

// GetDailyUptimeData retrieves aggregated daily uptime data
func (r *StatusRepositoryImpl) GetDailyUptimeData(ctx context.Context, serviceName value.ServiceName, days int) ([]*model.DailyUptimeData, error) {
	endDate := time.Now().UTC()
	startDate := endDate.AddDate(0, 0, -days)

	rows, err := r.queries.GetDailyUptimeData(ctx, sqlc.GetDailyUptimeDataParams{
		ServiceName: serviceName.String(),
		FromDate:    startDate,
		ToDate:      endDate,
	})
	if err != nil {
		return nil, err
	}

	data := make([]*model.DailyUptimeData, 0, len(rows))
	for _, row := range rows {
		var avgRespTime, p95RespTime *uint32
		if row.AvgResponseTimeMs.Valid {
			val := uint32(row.AvgResponseTimeMs.Int32)
			avgRespTime = &val
		}
		if row.P95ResponseTimeMs.Valid {
			val := uint32(row.P95ResponseTimeMs.Int32)
			p95RespTime = &val
		}

		// Parse uptime percentage from string (DECIMAL in MySQL)
		uptimePercentage := 0.0
		if _, err := fmt.Sscanf(row.UptimePercentage, "%f", &uptimePercentage); err != nil {
			uptimePercentage = 0.0
		}

		data = append(data, &model.DailyUptimeData{
			Date:              row.Date,
			UptimePercentage:  uptimePercentage,
			AvgResponseTimeMs: avgRespTime,
			P95ResponseTimeMs: p95RespTime,
			IncidentCount:     row.IncidentCount,
		})
	}

	return data, nil
}

// UpsertDailyUptime inserts or updates daily uptime statistics
func (r *StatusRepositoryImpl) UpsertDailyUptime(ctx context.Context, serviceName value.ServiceName, date time.Time, data *model.DailyUptimeData) error {
	var avgRespTime, p95RespTime sql.NullInt32
	if data.AvgResponseTimeMs != nil {
		avgRespTime = sql.NullInt32{Int32: int32(*data.AvgResponseTimeMs), Valid: true}
	}
	if data.P95ResponseTimeMs != nil {
		p95RespTime = sql.NullInt32{Int32: int32(*data.P95ResponseTimeMs), Valid: true}
	}

	// Calculate downtime minutes and incident count (placeholder values for now)
	downtimeMinutes := uint32(0)
	incidentCount := data.IncidentCount

	// Calculate total checks and successful checks based on uptime percentage
	totalChecks := uint32(1440) // Assuming checks every minute
	successfulChecks := uint32(float64(totalChecks) * data.UptimePercentage / 100.0)

	return r.queries.UpsertDailyUptime(ctx, sqlc.UpsertDailyUptimeParams{
		ServiceName:       serviceName.String(),
		Date:              date,
		TotalChecks:       totalChecks,
		SuccessfulChecks:  successfulChecks,
		UptimePercentage:  fmt.Sprintf("%.2f", data.UptimePercentage),
		AvgResponseTimeMs: avgRespTime,
		P95ResponseTimeMs: p95RespTime,
		DowntimeMinutes:   downtimeMinutes,
		IncidentCount:     incidentCount,
	})
}

// DeleteOldStatusChecks removes status checks older than the specified date
func (r *StatusRepositoryImpl) DeleteOldStatusChecks(ctx context.Context, olderThan time.Time) error {
	return r.queries.DeleteOldStatusChecks(ctx, olderThan)
}

// GetUptimeStats calculates uptime statistics for a service over a period
func (r *StatusRepositoryImpl) GetUptimeStats(ctx context.Context, serviceName value.ServiceName, hours int) (*model.UptimeStats, error) {
	row, err := r.queries.GetUptimeStats(ctx, sqlc.GetUptimeStatsParams{
		ServiceName: serviceName.String(),
		DATESUB:     hours,
	})
	if err != nil {
		return nil, err
	}

	stats := model.NewUptimeStats(serviceName)
	stats.TotalChecks = uint32(row.TotalChecks)

	// SuccessfulChecks is interface{}, need type assertion
	if successfulChecks, ok := row.SuccessfulChecks.(int64); ok {
		stats.SuccessfulChecks = uint32(successfulChecks)
	}

	stats.CalculateUptimePercentage()

	// AvgResponseTimeMs is int64, not nullable
	if row.AvgResponseTimeMs > 0 {
		val := uint32(row.AvgResponseTimeMs)
		stats.AvgResponseTimeMs = &val
	}

	return stats, nil
}

// rowToStatusCheck converts a SQLC row to a domain StatusCheck
func (r *StatusRepositoryImpl) rowToStatusCheck(row sqlc.ServiceStatusCheck) (*model.StatusCheck, error) {
	serviceName, err := value.NewServiceName(row.ServiceName)
	if err != nil {
		return nil, err
	}

	status, err := value.NewServiceStatus(string(row.Status))
	if err != nil {
		return nil, err
	}

	category, err := value.NewServiceCategory(string(row.ServiceCategory))
	if err != nil {
		return nil, err
	}

	var responseTimeMs *uint32
	if row.ResponseTimeMs.Valid {
		val := uint32(row.ResponseTimeMs.Int32)
		responseTimeMs = &val
	}

	var errorMsg *string
	if row.ErrorMessage.Valid {
		errorMsg = &row.ErrorMessage.String
	}

	var metadata map[string]interface{}
	if len(row.Metadata) > 0 {
		if err := json.Unmarshal(row.Metadata, &metadata); err != nil {
			metadata = make(map[string]interface{})
		}
	} else {
		metadata = make(map[string]interface{})
	}

	return &model.StatusCheck{
		CheckID:         uint64(row.CheckID),
		ServiceName:     serviceName,
		ServiceCategory: category,
		Status:          status,
		ResponseTimeMs:  responseTimeMs,
		ErrorMessage:    errorMsg,
		CheckedAt:       row.CheckedAt,
		Metadata:        metadata,
	}, nil
}
