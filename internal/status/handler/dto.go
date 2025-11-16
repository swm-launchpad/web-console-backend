package handler

import (
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
)

// StatusCheckDTO represents a status check response
type StatusCheckDTO struct {
	CheckID         uint64                 `json:"check_id"`
	ServiceName     string                 `json:"service_name"`
	ServiceCategory string                 `json:"service_category"`
	Status          string                 `json:"status"`
	ResponseTimeMs  *uint32                `json:"response_time_ms,omitempty"`
	ErrorMessage    *string                `json:"error_message,omitempty"`
	CheckedAt       time.Time              `json:"checked_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// UptimeStatsDTO represents uptime statistics response
type UptimeStatsDTO struct {
	ServiceName          string  `json:"service_name"`
	UptimePercentage     float64 `json:"uptime_percentage"`
	AvgResponseTimeMs    *uint32 `json:"avg_response_time_ms,omitempty"`
	P95ResponseTimeMs    *uint32 `json:"p95_response_time_ms,omitempty"`
	TotalChecks          uint32  `json:"total_checks"`
	SuccessfulChecks     uint32  `json:"successful_checks"`
	CurrentStreakSeconds uint32  `json:"current_streak_seconds"`
}

// DailyUptimeDTO represents daily uptime data response
type DailyUptimeDTO struct {
	Date              time.Time `json:"date"`
	UptimePercentage  float64   `json:"uptime_percentage"`
	AvgResponseTimeMs *uint32   `json:"avg_response_time_ms,omitempty"`
	P95ResponseTimeMs *uint32   `json:"p95_response_time_ms,omitempty"`
	IncidentCount     uint16    `json:"incident_count"`
}

// ToStatusCheckDTO converts domain StatusCheck to DTO
func ToStatusCheckDTO(check *model.StatusCheck) StatusCheckDTO {
	return StatusCheckDTO{
		CheckID:         check.CheckID,
		ServiceName:     check.ServiceName.String(),
		ServiceCategory: check.ServiceCategory.String(),
		Status:          check.Status.String(),
		ResponseTimeMs:  check.ResponseTimeMs,
		ErrorMessage:    check.ErrorMessage,
		CheckedAt:       check.CheckedAt,
		Metadata:        check.Metadata,
	}
}

// ToUptimeStatsDTO converts domain UptimeStats to DTO
func ToUptimeStatsDTO(stats *model.UptimeStats) UptimeStatsDTO {
	return UptimeStatsDTO{
		ServiceName:          stats.ServiceName.String(),
		UptimePercentage:     stats.UptimePercentage,
		AvgResponseTimeMs:    stats.AvgResponseTimeMs,
		P95ResponseTimeMs:    stats.P95ResponseTimeMs,
		TotalChecks:          stats.TotalChecks,
		SuccessfulChecks:     stats.SuccessfulChecks,
		CurrentStreakSeconds: stats.CurrentStreakSeconds,
	}
}

// ToDailyUptimeDTO converts domain DailyUptimeData to DTO
func ToDailyUptimeDTO(data *model.DailyUptimeData) DailyUptimeDTO {
	return DailyUptimeDTO{
		Date:              data.Date,
		UptimePercentage:  data.UptimePercentage,
		AvgResponseTimeMs: data.AvgResponseTimeMs,
		P95ResponseTimeMs: data.P95ResponseTimeMs,
		IncidentCount:     data.IncidentCount,
	}
}
