package value

import (
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

// TemplateStatus represents the status of a template
type TemplateStatus string

const (
	TemplateStatusActive     TemplateStatus = "active"
	TemplateStatusInactive   TemplateStatus = "inactive"
	TemplateStatusDeprecated TemplateStatus = "deprecated"
)

// NewTemplateStatus creates a new TemplateStatus value object
func NewTemplateStatus(status string) (TemplateStatus, error) {
	ts := TemplateStatus(status)
	if !ts.IsValid() {
		return "", containererrors.ErrInvalidTemplateConfig
	}
	return ts, nil
}

// IsValid checks if the template status is valid
func (ts TemplateStatus) IsValid() bool {
	switch ts {
	case TemplateStatusActive, TemplateStatusInactive, TemplateStatusDeprecated:
		return true
	default:
		return false
	}
}

// IsActive checks if the template is active
func (ts TemplateStatus) IsActive() bool {
	return ts == TemplateStatusActive
}

// String returns the string representation of the template status
func (ts TemplateStatus) String() string {
	return string(ts)
}

// Equals checks if two template statuses are equal
func (ts TemplateStatus) Equals(other TemplateStatus) bool {
	return ts == other
}
