package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	"github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/repository/sqlc"
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
	qtx := r.queriesWithContext(ctx)

	// Create project
	params := sqlc.CreateProjectParams{
		Name:                   project.Name(),
		Slug:                   project.Slug().String(),
		Fqdn:                   stringBoolToNullString(project.FQDN()),
		Status:                 sqlc.ProjectsStatus(project.Status()),
		Plan:                   stringBoolToNullString(project.Plan()),
		CpuLimit:               sql.NullInt32{Int32: int32(project.Limits().CPULimit()), Valid: true},
		MemoryLimit:            sql.NullInt32{Int32: int32(project.Limits().MemoryLimit()), Valid: true},
		DiskLimit:              sql.NullInt32{Int32: int32(project.Limits().DiskLimit()), Valid: true},
		TrafficLimit:           sql.NullInt64{Int64: int64(project.Limits().TrafficLimit()), Valid: true},
		ProjectOperationStatus: projectOperationStatusToDB(project.OperationStatus()),
		ActiveDeploymentID:     uintBoolToNullInt32(project.ActiveDeploymentID()),
		CreatedAt:              project.CreatedAt(),
		UpdatedAt:              sql.NullTime{Time: project.UpdatedAt(), Valid: !project.UpdatedAt().IsZero()},
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
	for _, user := range project.Users() {
		if user.IsActive() {
			userParams := sqlc.CreateProjectUserParams{
				ProjectID: uint32(project.ProjectID()),
				UserID:    uint32(user.UserID()),
				Role:      sqlc.ProjectUserRole(user.Role()),
				CreatedAt: user.CreatedAt(),
				UpdatedAt: sql.NullTime{Time: user.UpdatedAt(), Valid: !user.UpdatedAt().IsZero()},
			}
			if _, err := qtx.CreateProjectUser(ctx, userParams); err != nil {
				return projecterrors.ErrDatabaseOperation
			}
		}
	}

	return nil
}

func (r *projectRepository) Save(ctx context.Context, project *model.Project) error {
	qtx := r.queriesWithContext(ctx)

	// Update project
	params := sqlc.UpdateProjectParams{
		Name:                   project.Name(),
		Fqdn:                   stringBoolToNullString(project.FQDN()),
		Status:                 sqlc.ProjectsStatus(project.Status()),
		Plan:                   stringBoolToNullString(project.Plan()),
		CpuLimit:               sql.NullInt32{Int32: int32(project.Limits().CPULimit()), Valid: true},
		MemoryLimit:            sql.NullInt32{Int32: int32(project.Limits().MemoryLimit()), Valid: true},
		DiskLimit:              sql.NullInt32{Int32: int32(project.Limits().DiskLimit()), Valid: true},
		TrafficLimit:           sql.NullInt64{Int64: int64(project.Limits().TrafficLimit()), Valid: true},
		ProjectOperationStatus: projectOperationStatusToDB(project.OperationStatus()),
		ActiveDeploymentID:     uintBoolToNullInt32(project.ActiveDeploymentID()),
		UpdatedAt:              sql.NullTime{Time: project.UpdatedAt(), Valid: !project.UpdatedAt().IsZero()},
		ProjectID:              uint32(project.ProjectID()),
	}

	if _, err := qtx.UpdateProject(ctx, params); err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	// Load existing project users (including soft-deleted) to track changes
	existingUsers, err := qtx.GetAllProjectUsersByProjectID(ctx, uint32(project.ProjectID()))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return projecterrors.ErrDatabaseOperation
	}

	// Build map of existing users for quick lookup
	existingUserMap := make(map[uint]sqlc.ProjectUser)
	for _, u := range existingUsers {
		existingUserMap[uint(u.UserID)] = u
	}

	// Build map of current domain users
	currentUsers := project.Users()
	currentUserMap := make(map[uint]model.ProjectUser)
	for _, u := range currentUsers {
		currentUserMap[u.UserID()] = u
	}

	// Process changes: Compare existing DB users with current domain users
	for _, domainUser := range currentUsers {
		userID := domainUser.UserID()
		existingUser, exists := existingUserMap[userID]

		if !exists {
			// New user: INSERT
			userParams := sqlc.CreateProjectUserParams{
				ProjectID: uint32(project.ProjectID()),
				UserID:    uint32(userID),
				Role:      sqlc.ProjectUserRole(domainUser.Role()),
				CreatedAt: domainUser.CreatedAt(),
				UpdatedAt: sql.NullTime{Time: domainUser.UpdatedAt(), Valid: !domainUser.UpdatedAt().IsZero()},
			}
			if _, err := qtx.CreateProjectUser(ctx, userParams); err != nil {
				return projecterrors.ErrDatabaseOperation
			}
		} else {
			// Existing user: check for changes
			wasDeleted := existingUser.IsDeleted
			isDeleted := domainUser.IsDeleted()
			roleChanged := string(existingUser.Role) != string(domainUser.Role())

			if wasDeleted && !isDeleted {
				// Restore user: UPDATE (was deleted, now active)
				restoreParams := sqlc.RestoreProjectUserParams{
					Role:      sqlc.ProjectUserRole(domainUser.Role()),
					UpdatedAt: sql.NullTime{Time: domainUser.UpdatedAt(), Valid: !domainUser.UpdatedAt().IsZero()},
					ProjectID: uint32(project.ProjectID()),
					UserID:    uint32(userID),
				}
				if _, err := qtx.RestoreProjectUser(ctx, restoreParams); err != nil {
					return projecterrors.ErrDatabaseOperation
				}
			} else if !wasDeleted && isDeleted {
				// Soft delete user: UPDATE (was active, now deleted)
				deleteParams := sqlc.DeleteProjectUserParams{
					DeletedAt: toNullTime(domainUser.DeletedAt()),
					UpdatedAt: sql.NullTime{Time: domainUser.UpdatedAt(), Valid: !domainUser.UpdatedAt().IsZero()},
					ProjectID: uint32(project.ProjectID()),
					UserID:    uint32(userID),
				}
				if _, err := qtx.DeleteProjectUser(ctx, deleteParams); err != nil {
					return projecterrors.ErrDatabaseOperation
				}
			} else if !isDeleted && roleChanged {
				// Role changed: UPDATE
				updateParams := sqlc.UpdateProjectUserParams{
					Role:      sqlc.ProjectUserRole(domainUser.Role()),
					UpdatedAt: sql.NullTime{Time: domainUser.UpdatedAt(), Valid: !domainUser.UpdatedAt().IsZero()},
					ProjectID: uint32(project.ProjectID()),
					UserID:    uint32(userID),
				}
				if _, err := qtx.UpdateProjectUser(ctx, updateParams); err != nil {
					return projecterrors.ErrDatabaseOperation
				}
			}
			// else: no changes, do nothing
		}
	}

	// Handle soft delete
	if project.IsDeleted() {
		deleteParams := sqlc.DeleteProjectParams{
			DeletedAt: timeBoolToNullTime(project.DeletedAt()),
			UpdatedAt: sql.NullTime{Time: project.UpdatedAt(), Valid: !project.UpdatedAt().IsZero()},
			ProjectID: uint32(project.ProjectID()),
		}
		if _, err := qtx.DeleteProject(ctx, deleteParams); err != nil {
			return projecterrors.ErrDatabaseOperation
		}
	}

	return nil
}

func (r *projectRepository) FindByID(ctx context.Context, projectID uint) (*model.Project, error) {
	// Get project
	row, err := r.queriesWithContext(ctx).GetProjectByID(ctx, uint32(projectID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, projecterrors.ErrProjectNotFound
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Convert to domain model
	project, err := r.rowToDomainProject(
		row.ProjectID, row.Name, row.Slug, row.Fqdn, row.Status, row.Plan,
		row.CpuLimit, row.MemoryLimit, row.DiskLimit, row.TrafficLimit,
		row.ProjectOperationStatus, row.ActiveDeploymentID,
		row.CreatedAt, row.UpdatedAt, row.DeletedAt, row.IsDeleted,
	)
	if err != nil {
		return nil, err
	}

	// Load project users
	sqlcUsers, err := r.queriesWithContext(ctx).GetProjectUsersByProjectID(ctx, uint32(projectID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, projecterrors.ErrDatabaseOperation
	}

	if err := r.loadProjectUsers(project, sqlcUsers); err != nil {
		return nil, err
	}

	return project, nil
}

func (r *projectRepository) FindByIDForUpdate(ctx context.Context, projectID uint) (*model.Project, error) {
	// Get project with row lock
	row, err := r.queriesWithContext(ctx).GetProjectByIDForUpdate(ctx, uint32(projectID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, projecterrors.ErrProjectNotFound
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Convert to domain model
	project, err := r.rowToDomainProject(
		row.ProjectID, row.Name, row.Slug, row.Fqdn, row.Status, row.Plan,
		row.CpuLimit, row.MemoryLimit, row.DiskLimit, row.TrafficLimit,
		row.ProjectOperationStatus, row.ActiveDeploymentID,
		row.CreatedAt, row.UpdatedAt, row.DeletedAt, row.IsDeleted,
	)
	if err != nil {
		return nil, err
	}

	// Load project users
	sqlcUsers, err := r.queriesWithContext(ctx).GetProjectUsersByProjectID(ctx, uint32(projectID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, projecterrors.ErrDatabaseOperation
	}

	if err := r.loadProjectUsers(project, sqlcUsers); err != nil {
		return nil, err
	}

	return project, nil
}

func (r *projectRepository) FindByUserID(ctx context.Context, userID uint) ([]*model.Project, error) {
	qtx := r.queriesWithContext(ctx)

	sqlcProjects, err := qtx.ListProjectsByUserID(ctx, uint32(userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*model.Project{}, nil
		}
		return nil, projecterrors.ErrDatabaseOperation
	}

	if len(sqlcProjects) == 0 {
		return []*model.Project{}, nil
	}

	// Collect all project IDs
	projectIDs := make([]uint32, len(sqlcProjects))
	for i, p := range sqlcProjects {
		projectIDs[i] = p.ProjectID
	}

	// Load all project users in one query (eliminates N+1)
	allUsers, err := qtx.GetProjectUsersByProjectIDs(ctx, projectIDs)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Group users by project ID
	usersByProject := make(map[uint][]sqlc.ProjectUser)
	for _, user := range allUsers {
		pid := uint(user.ProjectID)
		usersByProject[pid] = append(usersByProject[pid], user)
	}

	// Convert to domain models
	projects := make([]*model.Project, 0, len(sqlcProjects))
	for _, row := range sqlcProjects {
		project, err := r.rowToDomainProject(
			row.ProjectID, row.Name, row.Slug, row.Fqdn, row.Status, row.Plan,
			row.CpuLimit, row.MemoryLimit, row.DiskLimit, row.TrafficLimit,
			row.ProjectOperationStatus, row.ActiveDeploymentID,
			row.CreatedAt, row.UpdatedAt, row.DeletedAt, row.IsDeleted,
		)
		if err != nil {
			return nil, err
		}

		// Add users for this project
		if users, ok := usersByProject[uint(row.ProjectID)]; ok {
			if err := r.loadProjectUsers(project, users); err != nil {
				return nil, err
			}
		}

		projects = append(projects, project)
	}

	return projects, nil
}

func (r *projectRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	result, err := r.queriesWithContext(ctx).ExistsBySlug(ctx, slug)
	if err != nil {
		return false, projecterrors.ErrDatabaseOperation
	}
	return result, nil
}

func (r *projectRepository) ExistsByNameAndUserID(ctx context.Context, name string, userID uint) (bool, error) {
	params := sqlc.ExistsByNameAndUserIDParams{
		Name:   name,
		UserID: uint32(userID),
	}
	result, err := r.queriesWithContext(ctx).ExistsByNameAndUserID(ctx, params)
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

	_, err := r.queriesWithContext(ctx).DeleteProject(ctx, params)
	if err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	return nil
}

// FindProjectsWithActiveOperations finds all projects that have ongoing operations
func (r *projectRepository) FindProjectsWithActiveOperations(ctx context.Context) ([]*model.Project, error) {
	qtx := r.queriesWithContext(ctx)

	rows, err := qtx.FindProjectsWithActiveOperations(ctx)
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	projects := make([]*model.Project, 0, len(rows))
	for _, row := range rows {
		project, err := r.rowToDomainProject(
			row.ProjectID, row.Name, row.Slug, row.Fqdn, row.Status, row.Plan,
			row.CpuLimit, row.MemoryLimit, row.DiskLimit, row.TrafficLimit,
			row.ProjectOperationStatus, row.ActiveDeploymentID,
			row.CreatedAt, row.UpdatedAt, row.DeletedAt, row.IsDeleted,
		)
		if err != nil {
			return nil, err
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// Helper methods

func (r *projectRepository) loadProjectUsers(project *model.Project, sqlcUsers []sqlc.ProjectUser) error {
	for _, sqlcUser := range sqlcUsers {
		user, err := r.toDomainProjectUser(sqlcUser)
		if err != nil {
			return err
		}
		if err := project.AddUser(user.UserID(), user.Role()); err != nil {
			// Any error here indicates data integrity issue
			// DB should have UNIQUE constraint to prevent duplicate users
			return projecterrors.ErrDatabaseOperation
		}
	}
	return nil
}

func (r *projectRepository) queriesWithContext(ctx context.Context) *sqlc.Queries {
	// Check if context has transaction
	if tx, ok := db.GetTx(ctx); ok && tx != nil {
		return r.queries.WithTx(tx)
	}

	return r.queries
}

func (r *projectRepository) rowToDomainProject(
	projectID uint32,
	name string,
	slugStr string,
	fqdn sql.NullString,
	status sqlc.ProjectsStatus,
	plan sql.NullString,
	cpuLimit sql.NullInt32,
	memoryLimit sql.NullInt32,
	diskLimit sql.NullInt32,
	trafficLimit sql.NullInt64,
	operationStatus sqlc.ProjectsProjectOperationStatus,
	activeDeploymentID sql.NullInt32,
	createdAt time.Time,
	updatedAt sql.NullTime,
	deletedAt sql.NullTime,
	isDeleted bool,
) (*model.Project, error) {
	slug, err := value.NewProjectSlug(slugStr)
	if err != nil {
		// DB에 저장된 slug가 유효하지 않은 경우 - 데이터 무결성 문제
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Reconstruct project from persistence
	// updatedAt is always valid in domain, use createdAt as fallback if missing in DB
	finalUpdatedAt := createdAt
	if updatedAt.Valid {
		finalUpdatedAt = updatedAt.Time
	}

	// Set resource limits (0 = unlimited)
	limits, err := value.NewResourceLimits(
		nullInt32ToUint32(cpuLimit),
		nullInt32ToUint32(memoryLimit),
		nullInt32ToUint32(diskLimit),
		nullInt64ToUint32(trafficLimit),
	)
	if err != nil {
		// DB에 저장된 리소스 제한이 유효하지 않은 경우 - 데이터 무결성 문제
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Convert operation status from DB
	domainOperationStatus := projectOperationStatusFromDB(operationStatus)

	// Convert active_deployment_id from DB
	var domainActiveDeploymentID *uint
	if activeDeploymentID.Valid {
		deploymentID := uint(activeDeploymentID.Int32)
		domainActiveDeploymentID = &deploymentID
	}

	project := model.ReconstructProject(
		uint(projectID),
		name,
		*slug,
		value.ProjectStatus(status),
		domainOperationStatus,
		domainActiveDeploymentID,
		*limits,
		createdAt,
		finalUpdatedAt,
		isDeleted,
		fromNullTime(deletedAt),
	)

	// Set fields
	if fqdn.Valid {
		if err := project.SetFQDN(fqdn.String); err != nil {
			// DB에 저장된 FQDN이 유효하지 않은 경우 - 데이터 무결성 문제
			return nil, projecterrors.ErrDatabaseOperation
		}
	}

	if plan.Valid {
		if err := project.SetPlan(plan.String); err != nil {
			// DB에 저장된 Plan이 유효하지 않은 경우 - 데이터 무결성 문제
			return nil, projecterrors.ErrDatabaseOperation
		}
	}

	if err := project.SetStatus(value.ProjectStatus(status)); err != nil {
		// DB에 저장된 Status가 유효하지 않은 경우 - 데이터 무결성 문제
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Handle soft delete
	if isDeleted {
		if err := project.SoftDelete(); err != nil {
			// DB 상태가 일관되지 않은 경우 - 데이터 무결성 문제
			return nil, projecterrors.ErrDatabaseOperation
		}
	}

	return project, nil
}

// projectOperationStatusFromDB converts sqlc ProjectsProjectOperationStatus to domain ProjectOperationStatus
func projectOperationStatusFromDB(status sqlc.ProjectsProjectOperationStatus) value.ProjectOperationStatus {
	switch status {
	case sqlc.ProjectsProjectOperationStatusNothing:
		return value.ProjectOperationStatusNothing
	case sqlc.ProjectsProjectOperationStatusBuilding:
		return value.ProjectOperationStatusBuilding
	case sqlc.ProjectsProjectOperationStatusDeploying:
		return value.ProjectOperationStatusDeploying
	default:
		return value.ProjectOperationStatusNothing
	}
}

// projectOperationStatusToDB converts domain ProjectOperationStatus to sqlc ProjectsProjectOperationStatus
func projectOperationStatusToDB(status value.ProjectOperationStatus) sqlc.ProjectsProjectOperationStatus {
	switch status {
	case value.ProjectOperationStatusNothing:
		return sqlc.ProjectsProjectOperationStatusNothing
	case value.ProjectOperationStatusBuilding:
		return sqlc.ProjectsProjectOperationStatusBuilding
	case value.ProjectOperationStatusDeploying:
		return sqlc.ProjectsProjectOperationStatusDeploying
	default:
		return sqlc.ProjectsProjectOperationStatusNothing
	}
}

func (r *projectRepository) toDomainProjectUser(sqlcUser sqlc.ProjectUser) (*model.ProjectUser, error) {
	user, err := model.NewProjectUser(
		uint(sqlcUser.ProjectID),
		uint(sqlcUser.UserID),
		value.ProjectUserRole(sqlcUser.Role),
	)
	if err != nil {
		// DB에 저장된 ProjectUser가 유효하지 않은 경우 - 데이터 무결성 문제
		return nil, projecterrors.ErrDatabaseOperation
	}

	if sqlcUser.IsDeleted {
		if err := user.SoftDelete(); err != nil {
			// DB 상태가 일관되지 않은 경우 - 데이터 무결성 문제
			return nil, projecterrors.ErrDatabaseOperation
		}
	}

	return user, nil
}

// Utility functions

func stringBoolToNullString(s string, ok bool) sql.NullString {
	if !ok {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func timeBoolToNullTime(t time.Time, ok bool) sql.NullTime {
	if !ok {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: t, Valid: true}
}

// nullInt32ToUint32 converts sql.NullInt32 to uint32
// Returns 0 if NULL (for backward compatibility with old data)
func nullInt32ToUint32(n sql.NullInt32) uint32 {
	if !n.Valid {
		return 0
	}
	return uint32(n.Int32)
}

// nullInt64ToUint32 converts sql.NullInt64 to uint32
// Returns 0 if NULL (for backward compatibility with old data)
func nullInt64ToUint32(n sql.NullInt64) uint32 {
	if !n.Valid {
		return 0
	}
	return uint32(n.Int64)
}

func fromNullTime(n sql.NullTime) *time.Time {
	if !n.Valid {
		return nil
	}
	return &n.Time
}

// uintBoolToNullInt32 converts (uint, bool) to sql.NullInt32
// Used for optional uint fields like ActiveDeploymentID
func uintBoolToNullInt32(val uint, ok bool) sql.NullInt32 {
	if !ok {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: int32(val), Valid: true}
}
