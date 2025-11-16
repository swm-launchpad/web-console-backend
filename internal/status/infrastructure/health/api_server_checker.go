package health

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/service"
)

// APIServerChecker checks the health of the API server (self)
type APIServerChecker struct{}

// NewAPIServerChecker creates a new APIServerChecker
func NewAPIServerChecker() service.HealthChecker {
	return &APIServerChecker{}
}

// ServiceName returns the service name
func (c *APIServerChecker) ServiceName() value.ServiceName {
	return value.ServiceAPIServer
}

// Check performs the health check
func (c *APIServerChecker) Check(ctx context.Context) (*model.StatusCheck, error) {
	// API server is always operational if this code is running
	responseTime := uint32(0)
	return model.NewStatusCheck(
		value.ServiceAPIServer,
		value.StatusOperational,
		&responseTime,
		nil,
	), nil
}
