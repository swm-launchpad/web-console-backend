package health

import (
	"context"
	"database/sql"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/service"
)

// MySQLChecker checks the health of MySQL database
type MySQLChecker struct {
	db *sql.DB
}

// NewMySQLChecker creates a new MySQLChecker
func NewMySQLChecker(db *sql.DB) service.HealthChecker {
	return &MySQLChecker{db: db}
}

// ServiceName returns the service name
func (c *MySQLChecker) ServiceName() value.ServiceName {
	return value.ServiceMySQL
}

// Check performs the health check
func (c *MySQLChecker) Check(ctx context.Context) (*model.StatusCheck, error) {
	start := time.Now()

	// Perform ping and SELECT 1 query
	if err := c.db.PingContext(ctx); err != nil {
		errorMsg := err.Error()
		return model.NewStatusCheck(
			value.ServiceMySQL,
			value.StatusDown,
			nil,
			&errorMsg,
		), nil
	}

	// Additional check: execute SELECT 1
	var result int
	if err := c.db.QueryRowContext(ctx, "SELECT 1").Scan(&result); err != nil {
		errorMsg := err.Error()
		return model.NewStatusCheck(
			value.ServiceMySQL,
			value.StatusDegraded,
			nil,
			&errorMsg,
		), nil
	}

	responseTime := uint32(time.Since(start).Milliseconds())
	return model.NewStatusCheck(
		value.ServiceMySQL,
		value.StatusOperational,
		&responseTime,
		nil,
	), nil
}
