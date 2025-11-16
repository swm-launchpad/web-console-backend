package cron

import (
	"context"
	"log"

	"github.com/robfig/cron/v3"
	"github.com/swm-launchpad/web-console-backend/internal/status/application"
)

// StatusCron manages scheduled tasks for status monitoring
type StatusCron struct {
	cron                 *cron.Cron
	performHealthChecks  *application.PerformHealthChecksUseCase
	calculateDailyUptime *application.CalculateDailyUptimeUseCase
	cleanupOldChecks     *application.CleanupOldChecksUseCase
	ctx                  context.Context
}

// NewStatusCron creates a new status cron scheduler
func NewStatusCron(
	performHealthChecks *application.PerformHealthChecksUseCase,
	calculateDailyUptime *application.CalculateDailyUptimeUseCase,
	cleanupOldChecks *application.CleanupOldChecksUseCase,
) *StatusCron {
	return &StatusCron{
		cron:                 cron.New(),
		performHealthChecks:  performHealthChecks,
		calculateDailyUptime: calculateDailyUptime,
		cleanupOldChecks:     cleanupOldChecks,
		ctx:                  context.Background(),
	}
}

// Start begins executing scheduled tasks
func (sc *StatusCron) Start() error {
	// Health checks every 1 minute
	_, err := sc.cron.AddFunc("*/1 * * * *", func() {
		if err := sc.performHealthChecks.Execute(sc.ctx); err != nil {
			log.Printf("Health check cron job failed: %v", err)
		}
	})
	if err != nil {
		return err
	}

	// Calculate daily uptime at 00:05 UTC every day
	_, err = sc.cron.AddFunc("5 0 * * *", func() {
		if err := sc.calculateDailyUptime.Execute(sc.ctx); err != nil {
			log.Printf("Daily uptime calculation cron job failed: %v", err)
		}
	})
	if err != nil {
		return err
	}

	// Cleanup old checks at 01:00 UTC every day
	_, err = sc.cron.AddFunc("0 1 * * *", func() {
		if err := sc.cleanupOldChecks.Execute(sc.ctx); err != nil {
			log.Printf("Cleanup old checks cron job failed: %v", err)
		}
	})
	if err != nil {
		return err
	}

	sc.cron.Start()
	log.Println("Status monitoring cron jobs started")
	return nil
}

// Stop stops all scheduled tasks
func (sc *StatusCron) Stop() {
	sc.cron.Stop()
	log.Println("Status monitoring cron jobs stopped")
}
