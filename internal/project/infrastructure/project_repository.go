package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/sqlc"
)

type projectRepository struct {
	db      sqlc.DBTX
	queries *sqlc.Queries
}

func NewProjectRepository(db sqlc.DBTX) repository.ProjectRepository {
	return &projectRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *projectRepository) Create(ctx context.Context, project *model.Project) error {
	// Check if we're already in a transaction
	var shouldCommit bool
	tx, existingTx := db.GetTx(ctx)
	if !existingTx || tx == nil {
		// No existing transaction, create our own
		var err error
		tx, err = r.beginTx(ctx)
		if err != nil {
			return err
		}
		defer func() {
			_ = tx.Rollback()
		}()
		shouldCommit = true
	}

	qtx := r.queriesWithContext(ctx, tx)

	// Create project
	params := sqlc.CreateProjectParams{
		Name:         project.GetName(),
		Slug:         project.GetSlug().String(),
		Fqdn:         toNullString(project.GetFQDN()),
		Status:       sqlc.ProjectsStatus(project.GetStatus()),
		Plan:         toNullString(project.GetPlan()),
		CpuLimit:     toNullInt32(project.GetLimits().GetCPULimit()),
		MemoryLimit:  toNullInt32(project.GetLimits().GetMemoryLimit()),
		DiskLimit:    toNullInt32(project.GetLimits().GetDiskLimit()),
		TrafficLimit: toNullInt64(project.GetLimits().GetTrafficLimit()),
		CreatedAt:    project.GetCreatedAt(),
		UpdatedAt:    toNullTime(project.GetUpdatedAt()),
	}

	result, err := qtx.CreateProject(ctx, params)
	if err != nil {
		if isDuplicateError(err) {
			return projecterrors.ErrSlugAlreadyExists
		}
		return projecterrors.ErrDatabaseOperation
	}

	// Get the auto-generated ID
	projectID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	project.SetProjectID(uint(projectID))

	// Save project users
	for _, user := range project.GetUsers() {
		if user.IsActive() {
			userParams := sqlc.CreateProjectUserParams{
				ProjectID: uint32(project.GetProjectID()),
				UserID:    uint32(user.GetUserID()),
				Role:      sqlc.ProjectUserRole(user.GetRole()),
				CreatedAt: user.GetCreatedAt(),
				UpdatedAt: toNullTime(user.GetUpdatedAt()),
			}
			if _, err := qtx.CreateProjectUser(ctx, userParams); err != nil {
				return projecterrors.ErrDatabaseOperation
			}
		}
	}

	// Volumes are now handled separately as their own aggregate

	if shouldCommit {
		return tx.Commit()
	}
	return nil
}

func (r *projectRepository) Save(ctx context.Context, project *model.Project) error {
	// Check if we're already in a transaction
	var shouldCommit bool
	tx, existingTx := db.GetTx(ctx)
	if !existingTx || tx == nil {
		// No existing transaction, create our own
		var err error
		tx, err = r.beginTx(ctx)
		if err != nil {
			return err
		}
		defer func() {
			_ = tx.Rollback()
		}()
		shouldCommit = true
	}

	qtx := r.queriesWithContext(ctx, tx)

	// Update project
	params := sqlc.UpdateProjectParams{
		Name:         project.GetName(),
		Fqdn:         toNullString(project.GetFQDN()),
		Status:       sqlc.ProjectsStatus(project.GetStatus()),
		Plan:         toNullString(project.GetPlan()),
		CpuLimit:     toNullInt32(project.GetLimits().GetCPULimit()),
		MemoryLimit:  toNullInt32(project.GetLimits().GetMemoryLimit()),
		DiskLimit:    toNullInt32(project.GetLimits().GetDiskLimit()),
		TrafficLimit: toNullInt64(project.GetLimits().GetTrafficLimit()),
		UpdatedAt:    toNullTime(project.GetUpdatedAt()),
		ProjectID:    uint32(project.GetProjectID()),
	}

	if _, err := qtx.UpdateProject(ctx, params); err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	// Hard delete all existing project users and recreate
	// We use hard delete to avoid unique constraint violations
	if _, err := qtx.HardDeleteProjectUsersByProjectID(ctx, uint32(project.GetProjectID())); err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	// Recreate project users
	for _, user := range project.GetUsers() {
		if user.IsActive() {
			userParams := sqlc.CreateProjectUserParams{
				ProjectID: uint32(project.GetProjectID()),
				UserID:    uint32(user.GetUserID()),
				Role:      sqlc.ProjectUserRole(user.GetRole()),
				CreatedAt: user.GetCreatedAt(),
				UpdatedAt: toNullTime(user.GetUpdatedAt()),
			}
			if _, err := qtx.CreateProjectUser(ctx, userParams); err != nil {
				return projecterrors.ErrDatabaseOperation
			}
		}
	}

	// Volumes are now handled separately as their own aggregate

	// Handle soft delete
	if project.IsSoftDeleted() {
		deleteParams := sqlc.DeleteProjectParams{
			DeletedAt: toNullTime(project.GetDeletedAt()),
			UpdatedAt: toNullTime(project.GetUpdatedAt()),
			ProjectID: uint32(project.GetProjectID()),
		}
		if _, err := qtx.DeleteProject(ctx, deleteParams); err != nil {
			return projecterrors.ErrDatabaseOperation
		}
	}

	if shouldCommit {
		return tx.Commit()
	}
	return nil
}

func (r *projectRepository) FindByID(ctx context.Context, projectID uint) (*model.Project, error) {
	// Get project
	sqlcProject, err := r.queriesWithContext(ctx, nil).GetProjectByID(ctx, uint32(projectID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, projecterrors.ErrProjectNotFound
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Convert to domain model
	project := r.toDomainProject(sqlcProject)

	// Load project users
	sqlcUsers, err := r.queriesWithContext(ctx, nil).GetProjectUsersByProjectID(ctx, uint32(projectID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, projecterrors.ErrDatabaseOperation
	}

	for _, sqlcUser := range sqlcUsers {
		user := r.toDomainProjectUser(sqlcUser)
		_ = project.AddUser(user.GetUserID(), user.GetRole())
	}

	// Volumes are now handled separately as their own aggregate

	return project, nil
}

func (r *projectRepository) FindBySlug(ctx context.Context, slug string) (*model.Project, error) {
	// Get project
	sqlcProject, err := r.queriesWithContext(ctx, nil).GetProjectBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, projecterrors.ErrProjectNotFound
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Convert to domain model and load associations
	return r.FindByID(ctx, uint(sqlcProject.ProjectID))
}

func (r *projectRepository) FindByUserID(ctx context.Context, userID uint) ([]*model.Project, error) {
	sqlcProjects, err := r.queriesWithContext(ctx, nil).ListProjectsByUserID(ctx, uint32(userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*model.Project{}, nil
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	projects := make([]*model.Project, 0, len(sqlcProjects))
	for _, sqlcProject := range sqlcProjects {
		// Load full project with associations
		project, err := r.FindByID(ctx, uint(sqlcProject.ProjectID))
		if err != nil {
			continue // Skip projects that can't be loaded
		}
		projects = append(projects, project)
	}

	return projects, nil
}

func (r *projectRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	result, err := r.queriesWithContext(ctx, nil).ExistsBySlug(ctx, slug)
	if err != nil {
		return false, projecterrors.ErrDatabaseOperation
	}
	return result, nil
}

func (r *projectRepository) Delete(ctx context.Context, projectID uint) error {
	now := time.Now()
	params := sqlc.DeleteProjectParams{
		DeletedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt: sql.NullTime{Time: now, Valid: true},
		ProjectID: uint32(projectID),
	}

	_, err := r.queriesWithContext(ctx, nil).DeleteProject(ctx, params)
	if err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	return nil
}

func (r *projectRepository) List(ctx context.Context, offset, limit int) ([]*model.Project, error) {
	params := sqlc.ListProjectsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	sqlcProjects, err := r.queriesWithContext(ctx, nil).ListProjects(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*model.Project{}, nil
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	projects := make([]*model.Project, 0, len(sqlcProjects))
	for _, sqlcProject := range sqlcProjects {
		// Load full project with associations
		project, err := r.FindByID(ctx, uint(sqlcProject.ProjectID))
		if err != nil {
			continue // Skip projects that can't be loaded
		}
		projects = append(projects, project)
	}

	return projects, nil
}

func (r *projectRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.queriesWithContext(ctx, nil).CountProjects(ctx)
	if err != nil {
		return 0, projecterrors.ErrDatabaseOperation
	}
	return count, nil
}

// Helper methods

func (r *projectRepository) queriesWithContext(ctx context.Context, tx *sql.Tx) *sqlc.Queries {
	if tx != nil {
		return r.queries.WithTx(tx)
	}

	// Check if context has transaction
	if ctxTx, ok := db.GetTx(ctx); ok && ctxTx != nil {
		return r.queries.WithTx(ctxTx)
	}

	return r.queries
}

func (r *projectRepository) beginTx(ctx context.Context) (*sql.Tx, error) {
	// Check if already in transaction
	if tx, ok := db.GetTx(ctx); ok && tx != nil {
		return tx, nil
	}

	// Get DB connection
	dbConn, ok := r.db.(*sql.DB)
	if !ok {
		return nil, errors.New("failed to get database connection")
	}

	// Begin new transaction
	return dbConn.Begin()
}

func (r *projectRepository) toDomainProject(sqlcProject sqlc.Project) *model.Project {
	slug, _ := model.NewProjectSlug(sqlcProject.Slug)

	// Reconstruct project from persistence
	project := model.ReconstructProject(
		uint(sqlcProject.ProjectID),
		sqlcProject.Name,
		*slug,
		model.ProjectStatus(sqlcProject.Status),
		sqlcProject.CreatedAt,
		fromNullTime(sqlcProject.UpdatedAt),
		sqlcProject.IsDeleted,
		fromNullTime(sqlcProject.DeletedAt),
	)

	// Set fields
	if sqlcProject.Fqdn.Valid {
		_ = project.SetFQDN(sqlcProject.Fqdn.String)
	}

	if sqlcProject.Plan.Valid {
		_ = project.UpdatePlan(sqlcProject.Plan.String)
	}

	_ = project.UpdateStatus(model.ProjectStatus(sqlcProject.Status))

	// Set resource limits
	limits, _ := model.NewResourceLimits(
		fromNullInt32(sqlcProject.CpuLimit),
		nil, // memoryRequest not in schema yet, will be added later
		fromNullInt32(sqlcProject.MemoryLimit),
		fromNullInt32(sqlcProject.DiskLimit),
		fromNullInt64(sqlcProject.TrafficLimit),
	)
	if limits != nil {
		_ = project.UpdateResourceLimits(*limits)
	}

	// Handle soft delete
	if sqlcProject.IsDeleted {
		_ = project.Delete()
	}

	return project
}

func (r *projectRepository) toDomainProjectUser(sqlcUser sqlc.ProjectUser) *model.ProjectUser {
	user, _ := model.NewProjectUser(
		uint(sqlcUser.ProjectID),
		uint(sqlcUser.UserID),
		model.ProjectUserRole(sqlcUser.Role),
	)

	if sqlcUser.IsDeleted {
		_ = user.SoftDelete()
	}

	return user
}

// Utility functions

func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

func toNullInt32(i *uint32) sql.NullInt32 {
	if i == nil {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: int32(*i), Valid: true}
}

func toNullInt64(i *uint64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}

func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func fromNullInt32(n sql.NullInt32) *uint32 {
	if !n.Valid || n.Int32 <= 0 {
		return nil
	}
	v := uint32(n.Int32)
	return &v
}

func fromNullInt64(n sql.NullInt64) *uint64 {
	if !n.Valid || n.Int64 <= 0 {
		return nil
	}
	v := uint64(n.Int64)
	return &v
}

func fromNullTime(n sql.NullTime) *time.Time {
	if !n.Valid {
		return nil
	}
	return &n.Time
}

// FindVolumeByID method removed - volumes are now handled by VolumeRepository

func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "unique")
}
