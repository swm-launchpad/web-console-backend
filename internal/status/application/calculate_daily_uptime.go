package application

import (
	"context"
	"log"
	"math"
	"sort"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/repository"
)

// CalculateDailyUptimeUseCase calculates and stores daily uptime statistics
type CalculateDailyUptimeUseCase struct {
	statusRepo   repository.StatusRepository
	incidentRepo repository.IncidentRepository
}

// NewCalculateDailyUptimeUseCase creates a new CalculateDailyUptimeUseCase
func NewCalculateDailyUptimeUseCase(statusRepo repository.StatusRepository, incidentRepo repository.IncidentRepository) *CalculateDailyUptimeUseCase {
	return &CalculateDailyUptimeUseCase{
		statusRepo:   statusRepo,
		incidentRepo: incidentRepo,
	}
}

// Execute calculates and stores daily uptime for all services for yesterday
func (uc *CalculateDailyUptimeUseCase) Execute(ctx context.Context) error {
	// Calculate for yesterday (00:00 - 23:59)
	now := time.Now().UTC()
	yesterday := now.AddDate(0, 0, -1)
	startOfDay := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Process each service
	allServices := value.AllServiceNames()
	for _, serviceName := range allServices {
		if err := uc.calculateForService(ctx, serviceName, startOfDay, endOfDay); err != nil {
			log.Printf("Failed to calculate daily uptime for %s: %v", serviceName, err)
			// Continue with other services even if one fails
		}
	}

	return nil
}

// calculateForService calculates daily uptime for a single service
func (uc *CalculateDailyUptimeUseCase) calculateForService(ctx context.Context, serviceName value.ServiceName, start, end time.Time) error {
	// Get all checks for the day
	checks, err := uc.statusRepo.GetStatusChecksByPeriod(ctx, serviceName, start, end)
	if err != nil {
		return err
	}

	if len(checks) == 0 {
		return nil // No data to calculate
	}

	// Calculate statistics
	totalChecks := uint32(len(checks))
	successfulChecks := uint32(0)
	var responseTimes []uint32
	incidentCount := uint32(0)
	lastStatus := value.StatusOperational

	for _, check := range checks {
		if check.Status == value.StatusOperational {
			successfulChecks++
		}
		if check.ResponseTimeMs != nil {
			responseTimes = append(responseTimes, *check.ResponseTimeMs)
		}

		// Detect incidents (status transition to degraded or down)
		if (check.Status == value.StatusDegraded || check.Status == value.StatusDown) &&
			lastStatus == value.StatusOperational {
			incidentCount++
		}
		lastStatus = check.Status
	}

	// Calculate uptime percentage
	uptimePercentage := 0.0
	if totalChecks > 0 {
		uptimePercentage = float64(successfulChecks) / float64(totalChecks) * 100.0
	}

	// Calculate average and P95 response time
	var avgResponseTime, p95ResponseTime *uint32
	if len(responseTimes) > 0 {
		sum := uint64(0)
		for _, rt := range responseTimes {
			sum += uint64(rt)
		}
		avg := uint32(sum / uint64(len(responseTimes)))
		avgResponseTime = &avg

		// Calculate P95
		sort.Slice(responseTimes, func(i, j int) bool {
			return responseTimes[i] < responseTimes[j]
		})
		p95Index := int(math.Ceil(float64(len(responseTimes))*0.95)) - 1
		if p95Index >= len(responseTimes) {
			p95Index = len(responseTimes) - 1
		}
		p95 := responseTimes[p95Index]
		p95ResponseTime = &p95
	}

	// Create daily uptime data
	dailyData := &model.DailyUptimeData{
		Date:              start,
		UptimePercentage:  uptimePercentage,
		AvgResponseTimeMs: avgResponseTime,
		P95ResponseTimeMs: p95ResponseTime,
		IncidentCount:     uint16(incidentCount),
	}

	// Store in database
	return uc.statusRepo.UpsertDailyUptime(ctx, serviceName, start, dailyData)
}
