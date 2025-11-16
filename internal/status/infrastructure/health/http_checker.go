package health

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/service"
)

// HTTPChecker checks HTTP endpoints
type HTTPChecker struct {
	serviceName value.ServiceName
	url         string
	client      *http.Client
	authHeader  map[string]string
}

// NewHTTPChecker creates a new HTTPChecker
func NewHTTPChecker(serviceName value.ServiceName, url string, timeout time.Duration, authHeader map[string]string) service.HealthChecker {
	return &HTTPChecker{
		serviceName: serviceName,
		url:         url,
		client: &http.Client{
			Timeout: timeout,
		},
		authHeader: authHeader,
	}
}

// ServiceName returns the service name
func (c *HTTPChecker) ServiceName() value.ServiceName {
	return c.serviceName
}

// Check performs the health check
func (c *HTTPChecker) Check(ctx context.Context) (*model.StatusCheck, error) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to create request: %v", err)
		return model.NewStatusCheck(c.serviceName, value.StatusDown, nil, &errorMsg), nil
	}

	// Add auth headers if provided
	for key, val := range c.authHeader {
		req.Header.Set(key, val)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		errorMsg := fmt.Sprintf("request failed: %v", err)
		return model.NewStatusCheck(c.serviceName, value.StatusDown, nil, &errorMsg), nil
	}
	defer func() { _ = resp.Body.Close() }()

	responseTime := uint32(time.Since(start).Milliseconds())

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return model.NewStatusCheck(c.serviceName, value.StatusOperational, &responseTime, nil), nil
	} else if resp.StatusCode >= 500 {
		errorMsg := fmt.Sprintf("server error: %d", resp.StatusCode)
		return model.NewStatusCheck(c.serviceName, value.StatusDown, &responseTime, &errorMsg), nil
	}

	errorMsg := fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
	return model.NewStatusCheck(c.serviceName, value.StatusDegraded, &responseTime, &errorMsg), nil
}
