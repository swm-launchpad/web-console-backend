package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
	"github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/repository/sqlc"
	"go.uber.org/zap"
)

type buildHistoryRepository struct {
	db      sqlc.DBTX
	queries *sqlc.Queries
	logger  logger.Logger
}

// NewBuildHistoryRepository creates a new build history repository instance
func NewBuildHistoryRepository(db sqlc.DBTX, log logger.Logger) repository.BuildHistoryRepository {
	return &buildHistoryRepository{
		db:      db,
		queries: sqlc.New(db),
		logger:  log,
	}
}

// Create creates a new build history record
// The build history ID will be set after successful creation
func (r *buildHistoryRepository) Create(ctx context.Context, b *build_history.BuildHistory) error {
	if b == nil {
		return projecterrors.ErrInvalidProjectData
	}

	r.logger.Info(ctx, "build history repository create started",
		zap.Uint("container_id", b.ContainerID()),
	)

	qtx := r.queriesWithContext(ctx)

	// Convert (value, bool) getters to pointers for nullable fields
	var summaryPtr *string
	if summary, exists := b.Summary(); exists {
		s := summary
		summaryPtr = &s
	}

	var tektonEventIDPtr *string
	if eventID, exists := b.TektonEventID(); exists {
		e := eventID
		tektonEventIDPtr = &e
	}

	var tektonPipelineRunNamePtr *string
	if runName, exists := b.TektonPipelineRunName(); exists {
		r := runName
		tektonPipelineRunNamePtr = &r
	}

	var gitCommitHashPtr *string
	if commitHash, exists := b.GitCommitHash(); exists {
		c := commitHash
		gitCommitHashPtr = &c
	}

	var startedAtPtr *time.Time
	if startedAt, exists := b.StartedAt(); exists {
		st := startedAt
		startedAtPtr = &st
	}

	var finishedAtPtr *time.Time
	if finishedAt, exists := b.FinishedAt(); exists {
		ft := finishedAt
		finishedAtPtr = &ft
	}

	result, err := qtx.CreateBuildHistory(ctx, sqlc.CreateBuildHistoryParams{
		ContainerID:           uint32(b.ContainerID()),
		Status:                buildHistoryStatusToDB(b.Status()),
		Summary:               stringPtrToNullString(summaryPtr),
		TektonEventID:         stringPtrToNullString(tektonEventIDPtr),
		TektonPipelineRunName: stringPtrToNullString(tektonPipelineRunNamePtr),
		GitCommitHash:         stringPtrToNullString(gitCommitHashPtr),
		CreatedAt:             b.CreatedAt(),
		StartedAt:             timePtrToNullTime(startedAtPtr),
		FinishedAt:            timePtrToNullTime(finishedAtPtr),
	})
	if err != nil {
		r.logger.Error(ctx, "build history repository create failed",
			zap.Uint("container_id", b.ContainerID()),
			zap.Error(err),
		)
		return projecterrors.ErrDatabaseOperation
	}

	id, err := result.LastInsertId()
	if err != nil {
		r.logger.Error(ctx, "build history repository create last insert id failed",
			zap.Uint("container_id", b.ContainerID()),
			zap.Error(err),
		)
		return projecterrors.ErrDatabaseOperation
	}

	b.SetBuildHistoryID(uint(id))
	r.logger.Info(ctx, "build history repository create completed",
		zap.Uint("build_history_id", b.BuildHistoryID),
		zap.Uint("container_id", b.ContainerID()),
	)
	return nil
}

// Save updates an existing build history record
func (r *buildHistoryRepository) Save(ctx context.Context, b *build_history.BuildHistory) error {
	if b == nil {
		return projecterrors.ErrInvalidProjectData
	}

	if b.BuildHistoryID == 0 {
		return projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	// Convert (value, bool) getters to pointers for nullable fields
	var summaryPtr *string
	if summary, exists := b.Summary(); exists {
		s := summary
		summaryPtr = &s
	}

	var tektonEventIDPtr *string
	if eventID, exists := b.TektonEventID(); exists {
		e := eventID
		tektonEventIDPtr = &e
	}

	var tektonPipelineRunNamePtr *string
	if runName, exists := b.TektonPipelineRunName(); exists {
		r := runName
		tektonPipelineRunNamePtr = &r
	}

	var gitCommitHashPtr *string
	if commitHash, exists := b.GitCommitHash(); exists {
		c := commitHash
		gitCommitHashPtr = &c
	}

	var startedAtPtr *time.Time
	if startedAt, exists := b.StartedAt(); exists {
		st := startedAt
		startedAtPtr = &st
	}

	var finishedAtPtr *time.Time
	if finishedAt, exists := b.FinishedAt(); exists {
		ft := finishedAt
		finishedAtPtr = &ft
	}

	result, err := qtx.UpdateBuildHistory(ctx, sqlc.UpdateBuildHistoryParams{
		Status:                buildHistoryStatusToDB(b.Status()),
		Summary:               stringPtrToNullString(summaryPtr),
		TektonEventID:         stringPtrToNullString(tektonEventIDPtr),
		TektonPipelineRunName: stringPtrToNullString(tektonPipelineRunNamePtr),
		GitCommitHash:         stringPtrToNullString(gitCommitHashPtr),
		StartedAt:             timePtrToNullTime(startedAtPtr),
		FinishedAt:            timePtrToNullTime(finishedAtPtr),
		BuildHistoryID:        uint32(b.BuildHistoryID),
	})
	if err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	// Check if any rows were affected
	// Note: MySQL returns 0 rows affected in two cases:
	// 1. Record doesn't exist (error case)
	// 2. Record exists but UPDATE with same values (idempotent, OK)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	if rowsAffected == 0 {
		// Verify if the build history exists to distinguish between case 1 and 2
		_, err := qtx.FindBuildHistoryByID(ctx, uint32(b.BuildHistoryID))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Case 1: Build history was deleted or never existed
				return projecterrors.ErrBuildHistoryNotFound
			}
			return projecterrors.ErrDatabaseOperation
		}
		// Case 2: Build history exists, idempotent update (same values)
		// This is a successful no-op, return nil
	}

	return nil
}

// FindByID finds a build history by its ID
func (r *buildHistoryRepository) FindByID(ctx context.Context, buildHistoryID uint) (*build_history.BuildHistory, error) {
	if buildHistoryID == 0 {
		return nil, projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	row, err := qtx.FindBuildHistoryByID(ctx, uint32(buildHistoryID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, projecterrors.ErrBuildHistoryNotFound
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	return r.rowToBuildHistoryModel(row.BuildHistoryID, row.ContainerID, row.Status, row.Summary,
		row.TektonEventID, row.TektonPipelineRunName, row.GitCommitHash, row.CreatedAt, row.StartedAt, row.FinishedAt)
}

// FindLatestByContainerID finds the most recent build history for a container
func (r *buildHistoryRepository) FindLatestByContainerID(ctx context.Context, containerID uint) (*build_history.BuildHistory, error) {
	if containerID == 0 {
		return nil, projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	row, err := qtx.FindLatestBuildHistoryByContainerID(ctx, uint32(containerID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, projecterrors.ErrBuildHistoryNotFound
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	return r.rowToBuildHistoryModel(row.BuildHistoryID, row.ContainerID, row.Status, row.Summary,
		row.TektonEventID, row.TektonPipelineRunName, row.GitCommitHash, row.CreatedAt, row.StartedAt, row.FinishedAt)
}

// FindByContainerID finds all build histories for a container with pagination
func (r *buildHistoryRepository) FindByContainerID(ctx context.Context, containerID uint, limit, offset int) ([]*build_history.BuildHistory, error) {
	if containerID == 0 {
		return nil, projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	sqlcBuildHistories, err := qtx.FindBuildHistoriesByContainerID(ctx, sqlc.FindBuildHistoriesByContainerIDParams{
		ContainerID: uint32(containerID),
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	buildHistories := make([]*build_history.BuildHistory, 0, len(sqlcBuildHistories))
	for _, row := range sqlcBuildHistories {
		b, err := r.rowToBuildHistoryModel(row.BuildHistoryID, row.ContainerID, row.Status, row.Summary,
			row.TektonEventID, row.TektonPipelineRunName, row.GitCommitHash, row.CreatedAt, row.StartedAt, row.FinishedAt)
		if err != nil {
			return nil, err
		}
		buildHistories = append(buildHistories, b)
	}

	return buildHistories, nil
}

// FindByTektonPipelineRunName finds a build history by its Tekton PipelineRun name
func (r *buildHistoryRepository) FindByTektonPipelineRunName(ctx context.Context, pipelineRunName string) (*build_history.BuildHistory, error) {
	if pipelineRunName == "" {
		return nil, projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	row, err := qtx.FindBuildHistoryByTektonPipelineRunName(ctx, toNullString(pipelineRunName))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, projecterrors.ErrBuildHistoryNotFound
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	return r.rowToBuildHistoryModel(row.BuildHistoryID, row.ContainerID, row.Status, row.Summary,
		row.TektonEventID, row.TektonPipelineRunName, row.GitCommitHash, row.CreatedAt, row.StartedAt, row.FinishedAt)
}

// FindActiveByContainerID finds all active (non-completed) build histories for a container
func (r *buildHistoryRepository) FindActiveByContainerID(ctx context.Context, containerID uint) ([]*build_history.BuildHistory, error) {
	if containerID == 0 {
		return nil, projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	sqlcBuildHistories, err := qtx.FindActiveBuildHistoriesByContainerID(ctx, uint32(containerID))
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	buildHistories := make([]*build_history.BuildHistory, 0, len(sqlcBuildHistories))
	for _, row := range sqlcBuildHistories {
		b, err := r.rowToBuildHistoryModel(row.BuildHistoryID, row.ContainerID, row.Status, row.Summary,
			row.TektonEventID, row.TektonPipelineRunName, row.GitCommitHash, row.CreatedAt, row.StartedAt, row.FinishedAt)
		if err != nil {
			return nil, err
		}
		buildHistories = append(buildHistories, b)
	}

	return buildHistories, nil
}

// rowToBuildHistoryModel converts sqlc query result row to domain BuildHistory model
func (r *buildHistoryRepository) rowToBuildHistoryModel(
	buildHistoryID uint32,
	containerID uint32,
	status sqlc.BuildHistoryStatus,
	summary sql.NullString,
	tektonEventID sql.NullString,
	tektonPipelineRunName sql.NullString,
	gitCommitHash sql.NullString,
	createdAt time.Time,
	startedAt sql.NullTime,
	finishedAt sql.NullTime,
) (*build_history.BuildHistory, error) {
	domainStatus := buildHistoryStatusFromDB(status)

	b, err := build_history.ReconstructBuildHistory(
		uint(buildHistoryID),
		uint(containerID),
		domainStatus,
		nullStringToStringPtr(summary),
		nullStringToStringPtr(tektonEventID),
		nullStringToStringPtr(tektonPipelineRunName),
		nullStringToStringPtr(gitCommitHash),
		createdAt,
		nullTimeToTimePtr(startedAt),
		nullTimeToTimePtr(finishedAt),
	)
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	return b, nil
}

// buildHistoryStatusToDB converts domain BuildHistoryStatus to sqlc BuildHistoryStatus
func buildHistoryStatusToDB(status build_history.BuildHistoryStatus) sqlc.BuildHistoryStatus {
	switch status {
	case build_history.BuildHistoryStatusUntracked:
		return sqlc.BuildHistoryStatusUntracked
	case build_history.BuildHistoryStatusBackendTriggerFailed:
		return sqlc.BuildHistoryStatusBackendTriggerFailed
	case build_history.BuildHistoryStatusBackendTrackingFailed:
		return sqlc.BuildHistoryStatusBackendTrackingFailed
	case build_history.BuildHistoryStatusBackendTrackingLost:
		return sqlc.BuildHistoryStatusBackendTrackingLost
	case build_history.BuildHistoryStatusRunning:
		return sqlc.BuildHistoryStatusRunning
	case build_history.BuildHistoryStatusSuccess:
		return sqlc.BuildHistoryStatusSuccess
	case build_history.BuildHistoryStatusFailed:
		return sqlc.BuildHistoryStatusFailed
	case build_history.BuildHistoryStatusCancelled:
		return sqlc.BuildHistoryStatusCancelled
	case build_history.BuildHistoryStatusSkipped:
		return sqlc.BuildHistoryStatusSkipped
	default:
		return sqlc.BuildHistoryStatusUntracked
	}
}

// buildHistoryStatusFromDB converts sqlc BuildHistoryStatus to domain BuildHistoryStatus
func buildHistoryStatusFromDB(status sqlc.BuildHistoryStatus) build_history.BuildHistoryStatus {
	switch status {
	case sqlc.BuildHistoryStatusUntracked:
		return build_history.BuildHistoryStatusUntracked
	case sqlc.BuildHistoryStatusBackendTriggerFailed:
		return build_history.BuildHistoryStatusBackendTriggerFailed
	case sqlc.BuildHistoryStatusBackendTrackingFailed:
		return build_history.BuildHistoryStatusBackendTrackingFailed
	case sqlc.BuildHistoryStatusBackendTrackingLost:
		return build_history.BuildHistoryStatusBackendTrackingLost
	case sqlc.BuildHistoryStatusRunning:
		return build_history.BuildHistoryStatusRunning
	case sqlc.BuildHistoryStatusSuccess:
		return build_history.BuildHistoryStatusSuccess
	case sqlc.BuildHistoryStatusFailed:
		return build_history.BuildHistoryStatusFailed
	case sqlc.BuildHistoryStatusCancelled:
		return build_history.BuildHistoryStatusCancelled
	case sqlc.BuildHistoryStatusSkipped:
		return build_history.BuildHistoryStatusSkipped
	default:
		return build_history.BuildHistoryStatusUntracked
	}
}

// queriesWithContext returns queries bound to transaction if available in context
func (r *buildHistoryRepository) queriesWithContext(ctx context.Context) *sqlc.Queries {
	if tx, ok := db.GetTx(ctx); ok && tx != nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}
