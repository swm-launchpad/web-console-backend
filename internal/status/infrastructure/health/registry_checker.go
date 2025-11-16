package health

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/service"
)

// RegistryChecker checks the health of the container registry
type RegistryChecker struct {
	httpChecker service.HealthChecker
}

// NewRegistryChecker creates a new RegistryChecker
func NewRegistryChecker(url string, timeout time.Duration) service.HealthChecker {
	return &RegistryChecker{
		httpChecker: NewHTTPChecker(value.ServiceRegistry, url+"/v2/", timeout, nil),
	}
}

// ServiceName returns the service name
func (c *RegistryChecker) ServiceName() value.ServiceName {
	return value.ServiceRegistry
}

// Check performs the health check
func (c *RegistryChecker) Check(ctx context.Context) (*model.StatusCheck, error) {
	return c.httpChecker.Check(ctx)
}
