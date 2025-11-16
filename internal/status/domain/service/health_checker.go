package service

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
)

// HealthChecker defines the interface for performing health checks on a service
type HealthChecker interface {
	// ServiceName returns the name of the service this checker monitors
	ServiceName() value.ServiceName

	// Check performs a health check and returns the result
	Check(ctx context.Context) (*model.StatusCheck, error)
}
