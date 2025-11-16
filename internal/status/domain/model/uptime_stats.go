package model

import (
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
)

// UptimeStats represents uptime statistics for a service over a period
type UptimeStats struct {
	ServiceName          value.ServiceName
	UptimePercentage     float64 // 0.00-100.00
	AvgResponseTimeMs    *uint32 // nullable
	P95ResponseTimeMs    *uint32 // nullable
	TotalChecks          uint32
	SuccessfulChecks     uint32
	CurrentStreakSeconds uint32 // Consecutive seconds of uptime
	DowntimeMinutes      uint32
	IncidentCount        uint16
}

// NewUptimeStats creates a new UptimeStats
func NewUptimeStats(serviceName value.ServiceName) *UptimeStats {
	return &UptimeStats{
		ServiceName:      serviceName,
		UptimePercentage: 0.0,
		TotalChecks:      0,
		SuccessfulChecks: 0,
		DowntimeMinutes:  0,
		IncidentCount:    0,
	}
}

// CalculateUptimePercentage calculates the uptime percentage
func (u *UptimeStats) CalculateUptimePercentage() {
	if u.TotalChecks == 0 {
		u.UptimePercentage = 0.0
		return
	}
	u.UptimePercentage = float64(u.SuccessfulChecks) / float64(u.TotalChecks) * 100.0
}

// DailyUptimeData represents daily uptime data for history
type DailyUptimeData struct {
	Date              time.Time
	UptimePercentage  float64
	AvgResponseTimeMs *uint32
	P95ResponseTimeMs *uint32
	IncidentCount     uint16
}
