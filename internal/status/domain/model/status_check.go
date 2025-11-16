package model

import (
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
)

// StatusCheck represents a single health check result for a service
type StatusCheck struct {
	CheckID         uint64
	ServiceName     value.ServiceName
	ServiceCategory value.ServiceCategory
	Status          value.ServiceStatus
	ResponseTimeMs  *uint32 // nullable
	ErrorMessage    *string // nullable
	CheckedAt       time.Time
	Metadata        map[string]interface{} // JSON metadata
}

// NewStatusCheck creates a new StatusCheck
func NewStatusCheck(
	serviceName value.ServiceName,
	status value.ServiceStatus,
	responseTimeMs *uint32,
	errorMessage *string,
) *StatusCheck {
	return &StatusCheck{
		ServiceName:     serviceName,
		ServiceCategory: value.GetCategoryForService(serviceName),
		Status:          status,
		ResponseTimeMs:  responseTimeMs,
		ErrorMessage:    errorMessage,
		CheckedAt:       time.Now().UTC(),
		Metadata:        make(map[string]interface{}),
	}
}

// IsHealthy returns true if the check indicates a healthy service
func (s *StatusCheck) IsHealthy() bool {
	return s.Status.IsHealthy()
}

// HasError returns true if the check has an error message
func (s *StatusCheck) HasError() bool {
	return s.ErrorMessage != nil && *s.ErrorMessage != ""
}
