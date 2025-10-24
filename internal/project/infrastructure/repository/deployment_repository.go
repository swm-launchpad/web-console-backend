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
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	"github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/repository/sqlc"
	"go.uber.org/zap"
)

type deploymentRepository struct {
	db      sqlc.DBTX
	queries *sqlc.Queries
	logger  logger.Logger
}

// NewDeploymentRepository creates a new deployment repository instance
func NewDeploymentRepository(db sqlc.DBTX, log logger.Logger) repository.DeploymentRepository {
	return &deploymentRepository{
		db:      db,
		queries: sqlc.New(db),
		logger:  log,
	}
}

// Create creates a new deployment record
// The deployment ID will be set after successful creation
func (r *deploymentRepository) Create(ctx context.Context, d *deployment.Deployment) error {
	if d == nil {
		return projecterrors.ErrInvalidProjectData
	}

	r.logger.Info(ctx, "deployment repository create started",
		zap.Uint("project_id", d.ProjectID()),
	)

	qtx := r.queriesWithContext(ctx)

	// Convert (value, bool) getters to pointers for nullable fields
	var summaryPtr *string
	if summary, exists := d.Summary(); exists {
		s := summary
		summaryPtr = &s
	}

	var tektonEventIDPtr *string
	if eventID, exists := d.TektonEventID(); exists {
		e := eventID
		tektonEventIDPtr = &e
	}

	var tektonPipelineRunNamePtr *string
	if runName, exists := d.TektonPipelineRunName(); exists {
		r := runName
		tektonPipelineRunNamePtr = &r
	}

	var startedAtPtr *time.Time
	if startedAt, exists := d.StartedAt(); exists {
		st := startedAt
		startedAtPtr = &st
	}

	var finishedAtPtr *time.Time
	if finishedAt, exists := d.FinishedAt(); exists {
		ft := finishedAt
		finishedAtPtr = &ft
	}

	result, err := qtx.CreateDeployment(ctx, sqlc.CreateDeploymentParams{
		ProjectID:             uint32(d.ProjectID()),
		Status:                deploymentStatusToDB(d.Status()),
		Summary:               stringPtrToNullString(summaryPtr),
		TektonEventID:         stringPtrToNullString(tektonEventIDPtr),
		TektonPipelineRunName: stringPtrToNullString(tektonPipelineRunNamePtr),
		CreatedAt:             d.CreatedAt(),
		StartedAt:             timePtrToNullTime(startedAtPtr),
		FinishedAt:            timePtrToNullTime(finishedAtPtr),
	})
	if err != nil {
		r.logger.Error(ctx, "deployment repository create failed",
			zap.Uint("project_id", d.ProjectID()),
			zap.Error(err),
		)
		return projecterrors.ErrDatabaseOperation
	}

	id, err := result.LastInsertId()
	if err != nil {
		r.logger.Error(ctx, "deployment repository create last insert id failed",
			zap.Uint("project_id", d.ProjectID()),
			zap.Error(err),
		)
		return projecterrors.ErrDatabaseOperation
	}

	d.SetDeploymentID(uint(id))
	r.logger.Info(ctx, "deployment repository create completed",
		zap.Uint("deployment_id", d.DeploymentID),
		zap.Uint("project_id", d.ProjectID()),
	)
	return nil
}

// Save updates an existing deployment record
func (r *deploymentRepository) Save(ctx context.Context, d *deployment.Deployment) error {
	if d == nil {
		return projecterrors.ErrInvalidProjectData
	}

	if d.DeploymentID == 0 {
		return projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	// Convert (value, bool) getters to pointers for nullable fields
	var summaryPtr *string
	if summary, exists := d.Summary(); exists {
		s := summary
		summaryPtr = &s
	}

	var tektonEventIDPtr *string
	if eventID, exists := d.TektonEventID(); exists {
		e := eventID
		tektonEventIDPtr = &e
	}

	var tektonPipelineRunNamePtr *string
	if runName, exists := d.TektonPipelineRunName(); exists {
		r := runName
		tektonPipelineRunNamePtr = &r
	}

	var startedAtPtr *time.Time
	if startedAt, exists := d.StartedAt(); exists {
		st := startedAt
		startedAtPtr = &st
	}

	var finishedAtPtr *time.Time
	if finishedAt, exists := d.FinishedAt(); exists {
		ft := finishedAt
		finishedAtPtr = &ft
	}

	result, err := qtx.UpdateDeployment(ctx, sqlc.UpdateDeploymentParams{
		Status:                deploymentStatusToDB(d.Status()),
		Summary:               stringPtrToNullString(summaryPtr),
		TektonEventID:         stringPtrToNullString(tektonEventIDPtr),
		TektonPipelineRunName: stringPtrToNullString(tektonPipelineRunNamePtr),
		StartedAt:             timePtrToNullTime(startedAtPtr),
		FinishedAt:            timePtrToNullTime(finishedAtPtr),
		DeploymentID:          uint32(d.DeploymentID),
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
		// Verify if the deployment exists to distinguish between case 1 and 2
		_, err := qtx.FindDeploymentByID(ctx, uint32(d.DeploymentID))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Case 1: Deployment was deleted or never existed
				return projecterrors.ErrDeploymentNotFound
			}
			return projecterrors.ErrDatabaseOperation
		}
		// Case 2: Deployment exists, idempotent update (same values)
		// This is a successful no-op, return nil
	}

	return nil
}

// FindByID finds a deployment by its ID
func (r *deploymentRepository) FindByID(ctx context.Context, deploymentID uint) (*deployment.Deployment, error) {
	if deploymentID == 0 {
		return nil, projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	row, err := qtx.FindDeploymentByID(ctx, uint32(deploymentID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, projecterrors.ErrDeploymentNotFound
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	return r.rowToDeploymentModel(row.DeploymentID, row.ProjectID, row.Status, row.Summary,
		row.TektonEventID, row.TektonPipelineRunName, row.CreatedAt, row.StartedAt, row.FinishedAt)
}

// FindLatestByProjectID finds the most recent deployment for a project
func (r *deploymentRepository) FindLatestByProjectID(ctx context.Context, projectID uint) (*deployment.Deployment, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	row, err := qtx.FindLatestDeploymentByProjectID(ctx, uint32(projectID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, projecterrors.ErrDeploymentNotFound
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	return r.rowToDeploymentModel(row.DeploymentID, row.ProjectID, row.Status, row.Summary,
		row.TektonEventID, row.TektonPipelineRunName, row.CreatedAt, row.StartedAt, row.FinishedAt)
}

// FindByProjectID finds all deployments for a project with pagination
func (r *deploymentRepository) FindByProjectID(ctx context.Context, projectID uint, limit, offset int) ([]*deployment.Deployment, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	sqlcDeployments, err := qtx.FindDeploymentsByProjectID(ctx, sqlc.FindDeploymentsByProjectIDParams{
		ProjectID: uint32(projectID),
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	deployments := make([]*deployment.Deployment, 0, len(sqlcDeployments))
	for _, row := range sqlcDeployments {
		d, err := r.rowToDeploymentModel(row.DeploymentID, row.ProjectID, row.Status, row.Summary,
			row.TektonEventID, row.TektonPipelineRunName, row.CreatedAt, row.StartedAt, row.FinishedAt)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, d)
	}

	return deployments, nil
}

// FindByTektonPipelineRunName finds a deployment by its Tekton PipelineRun name
func (r *deploymentRepository) FindByTektonPipelineRunName(ctx context.Context, pipelineRunName string) (*deployment.Deployment, error) {
	if pipelineRunName == "" {
		return nil, projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	row, err := qtx.FindDeploymentByTektonPipelineRunName(ctx, toNullString(pipelineRunName))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, projecterrors.ErrDeploymentNotFound
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	return r.rowToDeploymentModel(row.DeploymentID, row.ProjectID, row.Status, row.Summary,
		row.TektonEventID, row.TektonPipelineRunName, row.CreatedAt, row.StartedAt, row.FinishedAt)
}

// FindActiveDeploymentsByProjectID finds all active (non-completed) deployments for a project
func (r *deploymentRepository) FindActiveDeploymentsByProjectID(ctx context.Context, projectID uint) ([]*deployment.Deployment, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	sqlcDeployments, err := qtx.FindActiveDeploymentsByProjectID(ctx, uint32(projectID))
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	deployments := make([]*deployment.Deployment, 0, len(sqlcDeployments))
	for _, row := range sqlcDeployments {
		d, err := r.rowToDeploymentModel(row.DeploymentID, row.ProjectID, row.Status, row.Summary,
			row.TektonEventID, row.TektonPipelineRunName, row.CreatedAt, row.StartedAt, row.FinishedAt)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, d)
	}

	return deployments, nil
}

// rowToDeploymentModel converts sqlc query result row to domain Deployment model
func (r *deploymentRepository) rowToDeploymentModel(
	deploymentID uint32,
	projectID uint32,
	status sqlc.DeploymentsStatus,
	summary sql.NullString,
	tektonEventID sql.NullString,
	tektonPipelineRunName sql.NullString,
	createdAt time.Time,
	startedAt sql.NullTime,
	finishedAt sql.NullTime,
) (*deployment.Deployment, error) {
	domainStatus := deploymentStatusFromDB(status)

	d, err := deployment.ReconstructDeployment(
		uint(deploymentID),
		uint(projectID),
		domainStatus,
		nullStringToStringPtr(summary),
		nullStringToStringPtr(tektonEventID),
		nullStringToStringPtr(tektonPipelineRunName),
		createdAt,
		nullTimeToTimePtr(startedAt),
		nullTimeToTimePtr(finishedAt),
	)
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	return d, nil
}

// deploymentStatusToDB converts domain DeploymentStatus to sqlc DeploymentsStatus
func deploymentStatusToDB(status deployment.DeploymentStatus) sqlc.DeploymentsStatus {
	switch status {
	case deployment.DeploymentStatusUntracked:
		return sqlc.DeploymentsStatusUntracked
	case deployment.DeploymentStatusBackendTriggerFailed:
		return sqlc.DeploymentsStatusBackendTriggerFailed
	case deployment.DeploymentStatusBackendTrackingFailed:
		return sqlc.DeploymentsStatusBackendTrackingFailed
	case deployment.DeploymentStatusBackendTrackingLost:
		return sqlc.DeploymentsStatusBackendTrackingLost
	case deployment.DeploymentStatusRunning:
		return sqlc.DeploymentsStatusRunning
	case deployment.DeploymentStatusSuccess:
		return sqlc.DeploymentsStatusSuccess
	case deployment.DeploymentStatusFailed:
		return sqlc.DeploymentsStatusFailed
	case deployment.DeploymentStatusCancelled:
		return sqlc.DeploymentsStatusCancelled
	default:
		return sqlc.DeploymentsStatusUntracked
	}
}

// deploymentStatusFromDB converts sqlc DeploymentsStatus to domain DeploymentStatus
func deploymentStatusFromDB(status sqlc.DeploymentsStatus) deployment.DeploymentStatus {
	switch status {
	case sqlc.DeploymentsStatusUntracked:
		return deployment.DeploymentStatusUntracked
	case sqlc.DeploymentsStatusBackendTriggerFailed:
		return deployment.DeploymentStatusBackendTriggerFailed
	case sqlc.DeploymentsStatusBackendTrackingFailed:
		return deployment.DeploymentStatusBackendTrackingFailed
	case sqlc.DeploymentsStatusBackendTrackingLost:
		return deployment.DeploymentStatusBackendTrackingLost
	case sqlc.DeploymentsStatusRunning:
		return deployment.DeploymentStatusRunning
	case sqlc.DeploymentsStatusSuccess:
		return deployment.DeploymentStatusSuccess
	case sqlc.DeploymentsStatusFailed:
		return deployment.DeploymentStatusFailed
	case sqlc.DeploymentsStatusCancelled:
		return deployment.DeploymentStatusCancelled
	default:
		return deployment.DeploymentStatusUntracked
	}
}

// queriesWithContext returns queries bound to transaction if available in context
func (r *deploymentRepository) queriesWithContext(ctx context.Context) *sqlc.Queries {
	if tx, ok := db.GetTx(ctx); ok && tx != nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}
