package health

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/service"
)

// LokiChecker checks the health of Loki logging service
type LokiChecker struct {
	httpChecker service.HealthChecker
}

// NewLokiChecker creates a new LokiChecker
func NewLokiChecker(url string, timeout time.Duration) service.HealthChecker {
	return &LokiChecker{
		httpChecker: NewHTTPChecker(value.ServiceLoki, url+"/ready", timeout, nil),
	}
}

// ServiceName returns the service name
func (c *LokiChecker) ServiceName() value.ServiceName {
	return value.ServiceLoki
}

// Check performs the health check
func (c *LokiChecker) Check(ctx context.Context) (*model.StatusCheck, error) {
	return c.httpChecker.Check(ctx)
}
