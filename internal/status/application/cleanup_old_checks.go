package application

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/repository"
)

// CleanupOldChecksUseCase removes old status check data
type CleanupOldChecksUseCase struct {
	statusRepo repository.StatusRepository
}

// NewCleanupOldChecksUseCase creates a new CleanupOldChecksUseCase
func NewCleanupOldChecksUseCase(statusRepo repository.StatusRepository) *CleanupOldChecksUseCase {
	return &CleanupOldChecksUseCase{
		statusRepo: statusRepo,
	}
}

// Execute removes status checks older than the retention period (30 days)
func (uc *CleanupOldChecksUseCase) Execute(ctx context.Context) error {
	retentionDays := 30
	olderThan := time.Now().UTC().AddDate(0, 0, -retentionDays)

	return uc.statusRepo.DeleteOldStatusChecks(ctx, olderThan)
}
