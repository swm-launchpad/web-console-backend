package value

import (
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

// MonthlyMetrics represents container monthly usage metrics
type MonthlyMetrics struct {
	buildTime  *uint   // Total build time in seconds for the current month
	buildCount *uint   // Total number of builds for the current month
	uptime     *string // Uptime percentage as string (e.g., "99.9%")
}

// NewMonthlyMetrics creates a new MonthlyMetrics value object
func NewMonthlyMetrics(buildTime, buildCount *uint, uptime *string) (MonthlyMetrics, error) {
	// Note: buildTime and buildCount are unsigned, so no need to check < 0

	// Validate uptime format (should be a percentage string or number between 0-100)
	if uptime != nil && *uptime != "" {
		// Basic validation - more sophisticated parsing can be added if needed
		// For now, just ensure it's not too long (max 10 chars as per schema)
		if len(*uptime) > 10 {
			return MonthlyMetrics{}, containererrors.ErrInvalidUptime
		}
	}

	return MonthlyMetrics{
		buildTime:  buildTime,
		buildCount: buildCount,
		uptime:     uptime,
	}, nil
}

// BuildTime returns the total build time in seconds for the current month
func (m MonthlyMetrics) BuildTime() *uint {
	return m.buildTime
}

// BuildCount returns the total number of builds for the current month
func (m MonthlyMetrics) BuildCount() *uint {
	return m.buildCount
}

// Uptime returns the uptime percentage as a string
func (m MonthlyMetrics) Uptime() *string {
	return m.uptime
}

// Equals checks if two MonthlyMetrics are equal
func (m MonthlyMetrics) Equals(other MonthlyMetrics) bool {
	buildTimeEqual := (m.buildTime == nil && other.buildTime == nil) ||
		(m.buildTime != nil && other.buildTime != nil && *m.buildTime == *other.buildTime)

	buildCountEqual := (m.buildCount == nil && other.buildCount == nil) ||
		(m.buildCount != nil && other.buildCount != nil && *m.buildCount == *other.buildCount)

	uptimeEqual := (m.uptime == nil && other.uptime == nil) ||
		(m.uptime != nil && other.uptime != nil && *m.uptime == *other.uptime)

	return buildTimeEqual && buildCountEqual && uptimeEqual
}
