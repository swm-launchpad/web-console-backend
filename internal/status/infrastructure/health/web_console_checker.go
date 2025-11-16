package health

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/service"
)

// WebConsoleChecker checks the health of the Web Console frontend
type WebConsoleChecker struct {
	httpChecker service.HealthChecker
}

// NewWebConsoleChecker creates a new WebConsoleChecker
func NewWebConsoleChecker(url string, timeout time.Duration) service.HealthChecker {
	return &WebConsoleChecker{
		httpChecker: NewHTTPChecker(value.ServiceWebConsole, url+"/health", timeout, nil),
	}
}

// ServiceName returns the service name
func (c *WebConsoleChecker) ServiceName() value.ServiceName {
	return value.ServiceWebConsole
}

// Check performs the health check
func (c *WebConsoleChecker) Check(ctx context.Context) (*model.StatusCheck, error) {
	return c.httpChecker.Check(ctx)
}
