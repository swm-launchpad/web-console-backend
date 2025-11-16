package application

import (
	"context"
	"log"
	"sync"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/repository"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/service"
)

// PerformHealthChecksUseCase executes all health checkers and stores results
type PerformHealthChecksUseCase struct {
	statusRepo repository.StatusRepository
	checkers   []service.HealthChecker
}

// NewPerformHealthChecksUseCase creates a new PerformHealthChecksUseCase
func NewPerformHealthChecksUseCase(statusRepo repository.StatusRepository, checkers []service.HealthChecker) *PerformHealthChecksUseCase {
	return &PerformHealthChecksUseCase{
		statusRepo: statusRepo,
		checkers:   checkers,
	}
}

// Execute performs health checks for all services concurrently
func (uc *PerformHealthChecksUseCase) Execute(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(uc.checkers))

	for _, checker := range uc.checkers {
		wg.Add(1)
		go func(c service.HealthChecker) {
			defer wg.Done()

			// Perform health check
			check, err := c.Check(ctx)
			if err != nil {
				log.Printf("Health check failed for %s: %v", c.ServiceName(), err)
				errChan <- err
				return
			}

			// Store the result
			if err := uc.statusRepo.CreateStatusCheck(ctx, check); err != nil {
				log.Printf("Failed to store health check for %s: %v", c.ServiceName(), err)
				errChan <- err
			}
		}(checker)
	}

	wg.Wait()
	close(errChan)

	// Return first error if any occurred
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}
