package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	"github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/repository/sqlc"
)

type deploymentRepository struct {
	db      sqlc.DBTX
	queries *sqlc.Queries
}

// NewDeploymentRepository creates a new deployment repository instance
func NewDeploymentRepository(db sqlc.DBTX) repository.DeploymentRepository {
	return &deploymentRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

// Create creates a new deployment record
// The deployment ID will be set after successful creation
func (r *deploymentRepository) Create(ctx context.Context, d *deployment.Deployment) error {
	if d == nil {
		return projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	// TODO: After sqlc regeneration, use TektonEventID and TektonPipelineRunName
	// For now, use TektonRef for backward compatibility
	tektonRef := d.TektonPipelineRunName()
	if tektonRef == "" {
		tektonRef = d.TektonEventID()
	}

	result, err := qtx.CreateDeployment(ctx, sqlc.CreateDeploymentParams{
		ProjectID:  uint32(d.ProjectID()),
		Status:     deploymentStatusToDB(d.Status()),
		Summary:    toNullString(d.Summary()),
		TektonRef:  toNullString(tektonRef),
		CreatedAt:  d.CreatedAt(),
		StartedAt:  timeToNullTime(d.StartedAt()),
		FinishedAt: timeToNullTime(d.FinishedAt()),
	})
	if err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	id, err := result.LastInsertId()
	if err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	d.SetDeploymentID(uint(id))
	return nil
}

// Save updates an existing deployment record
func (r *deploymentRepository) Save(ctx context.Context, d *deployment.Deployment) error {
	if d == nil {
		return projecterrors.ErrInvalidProjectData
	}

	if d.DeploymentID() == 0 {
		return projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	// TODO: After sqlc regeneration, use TektonEventID and TektonPipelineRunName
	// For now, use TektonRef for backward compatibility
	tektonRef := d.TektonPipelineRunName()
	if tektonRef == "" {
		tektonRef = d.TektonEventID()
	}

	result, err := qtx.UpdateDeployment(ctx, sqlc.UpdateDeploymentParams{
		Status:       deploymentStatusToDB(d.Status()),
		Summary:      toNullString(d.Summary()),
		TektonRef:    toNullString(tektonRef),
		StartedAt:    timeToNullTime(d.StartedAt()),
		FinishedAt:   timeToNullTime(d.FinishedAt()),
		DeploymentID: uint32(d.DeploymentID()),
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
		_, err := qtx.FindDeploymentByID(ctx, uint32(d.DeploymentID()))
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

	sqlcDeployment, err := qtx.FindDeploymentByID(ctx, uint32(deploymentID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, projecterrors.ErrDeploymentNotFound
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	return r.toDeploymentModel(sqlcDeployment)
}

// FindLatestByProjectID finds the most recent deployment for a project
func (r *deploymentRepository) FindLatestByProjectID(ctx context.Context, projectID uint) (*deployment.Deployment, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	sqlcDeployment, err := qtx.FindLatestDeploymentByProjectID(ctx, uint32(projectID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, projecterrors.ErrDeploymentNotFound
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	return r.toDeploymentModel(sqlcDeployment)
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
	for _, sqlcDeployment := range sqlcDeployments {
		d, err := r.toDeploymentModel(sqlcDeployment)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, d)
	}

	return deployments, nil
}

// toDeploymentModel converts a sqlc Deployment to a domain Deployment model
func (r *deploymentRepository) toDeploymentModel(sqlcDeployment sqlc.Deployment) (*deployment.Deployment, error) {
	status := deploymentStatusFromDB(sqlcDeployment.Status)

	// TODO: After sqlc regeneration, tekton_ref will be split into tekton_event_id and tekton_pipeline_run_name
	// For now, use tekton_ref for both fields
	tektonRef := fromNullString(sqlcDeployment.TektonRef)

	d, err := deployment.ReconstructDeployment(
		uint(sqlcDeployment.DeploymentID),
		uint(sqlcDeployment.ProjectID),
		status,
		fromNullString(sqlcDeployment.Summary),
		tektonRef, // tektonEventID
		tektonRef, // tektonPipelineRunName
		sqlcDeployment.CreatedAt,
		nullTimeToTime(sqlcDeployment.StartedAt),
		nullTimeToTime(sqlcDeployment.FinishedAt),
	)
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	return d, nil
}

// deploymentStatusToDB converts domain DeploymentStatus to sqlc DeploymentsStatus
// TODO: After sqlc regeneration, map to new status enums directly
func deploymentStatusToDB(status deployment.DeploymentStatus) sqlc.DeploymentsStatus {
	switch status {
	case deployment.DeploymentStatusUntracked:
		return sqlc.DeploymentsStatusPending // Temporary mapping
	case deployment.DeploymentStatusBackendTriggerFailed:
		return sqlc.DeploymentsStatusFailed // Temporary mapping
	case deployment.DeploymentStatusBackendTrackingFailed:
		return sqlc.DeploymentsStatusFailed // Temporary mapping
	case deployment.DeploymentStatusBackendTrackingLost:
		return sqlc.DeploymentsStatusFailed // Temporary mapping
	case deployment.DeploymentStatusRunning:
		return sqlc.DeploymentsStatusRunning
	case deployment.DeploymentStatusSuccess:
		return sqlc.DeploymentsStatusSuccess
	case deployment.DeploymentStatusFailed:
		return sqlc.DeploymentsStatusFailed
	default:
		return sqlc.DeploymentsStatusPending
	}
}

// deploymentStatusFromDB converts sqlc DeploymentsStatus to domain DeploymentStatus
// TODO: After sqlc regeneration, map from new status enums directly
func deploymentStatusFromDB(status sqlc.DeploymentsStatus) deployment.DeploymentStatus {
	switch status {
	case sqlc.DeploymentsStatusPending:
		return deployment.DeploymentStatusUntracked // Temporary mapping
	case sqlc.DeploymentsStatusRunning:
		return deployment.DeploymentStatusRunning
	case sqlc.DeploymentsStatusSuccess:
		return deployment.DeploymentStatusSuccess
	case sqlc.DeploymentsStatusFailed:
		return deployment.DeploymentStatusFailed
	case sqlc.DeploymentsStatusCancelled:
		return deployment.DeploymentStatusFailed // Temporary mapping
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
