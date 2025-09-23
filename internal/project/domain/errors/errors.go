// Package errors defines domain-specific errors for the project bounded context.
// These errors represent business-level meanings without transport-specific codes.
package errors

import "errors"

// Repository errors - errors that occur at the data access layer
var (
	ErrProjectNotFound      = errors.New("project not found")
	ErrProjectAlreadyExists = errors.New("project already exists")
	ErrProjectUserNotFound  = errors.New("project user not found")
	ErrSlugAlreadyExists    = errors.New("slug already exists")
)

// Domain errors - business logic violations and domain rule errors
var (
	ErrInvalidProjectData         = errors.New("invalid project data")
	ErrProjectNotActive           = errors.New("project is not active")
	ErrCannotActivateDeleted      = errors.New("cannot activate deleted project")
	ErrCannotDeleteProject        = errors.New("cannot delete project")
	ErrInvalidStatusTransition    = errors.New("invalid status transition")
	ErrCannotModifyDeletedProject = errors.New("cannot modify deleted project")
)

// Permission errors - access control and authorization errors
var (
	ErrPermissionDenied      = errors.New("permission denied")
	ErrOwnerRequired         = errors.New("owner permission required")
	ErrCannotRemoveLastOwner = errors.New("cannot remove last owner")
	ErrUserAlreadyInProject  = errors.New("user already in project")
	ErrUserNotInProject      = errors.New("user not in project")
	ErrInvalidUserRole       = errors.New("invalid user role")
)

// Validation errors - input validation and data integrity errors
var (
	ErrNameRequired      = errors.New("project name is required")
	ErrSlugRequired      = errors.New("project slug is required")
	ErrInvalidSlug       = errors.New("invalid project slug")
	ErrSlugTooShort      = errors.New("slug must be at least 3 characters long")
	ErrSlugTooLong       = errors.New("slug must not exceed 63 characters")
	ErrSlugInvalidFormat = errors.New("slug can only contain lowercase letters, numbers, and hyphens")
	ErrSlugReserved      = errors.New("slug is reserved")
	ErrInvalidProjectID  = errors.New("invalid project ID")
	ErrInvalidUserID     = errors.New("invalid user ID")
	ErrValidationFailed  = errors.New("validation failed")
	ErrInvalidFormat     = errors.New("invalid format")
	ErrMissingField      = errors.New("required field is missing")
	ErrOwnerIDRequired   = errors.New("owner ID is required")
)

// Resource limit errors - errors related to resource constraints
var (
	ErrInvalidResourceLimits = errors.New("invalid resource limits")
	ErrCPULimitNegative      = errors.New("CPU limit cannot be negative")
	ErrMemoryLimitNegative   = errors.New("memory limit cannot be negative")
	ErrDiskLimitNegative     = errors.New("disk limit cannot be negative")
	ErrTrafficLimitNegative  = errors.New("traffic limit cannot be negative")
	ErrResourceLimitExceeded = errors.New("resource limit exceeded")
	ErrPlanLimitExceeded     = errors.New("plan limit exceeded")
	ErrIncompatiblePlan      = errors.New("resource limits incompatible with plan")
)

// Plan errors - errors related to project plans
var (
	ErrInvalidPlan          = errors.New("invalid plan")
	ErrPlanDowngradeBlocked = errors.New("cannot downgrade plan due to current resource usage")
	ErrPlanNotFound         = errors.New("plan not found")
)

// Volume errors - errors related to volume management
var (
	ErrVolumeNotFound      = errors.New("volume not found")
	ErrVolumeNameRequired  = errors.New("volume name is required")
	ErrInvalidCapacity     = errors.New("invalid volume capacity")
	ErrDuplicateVolumeName = errors.New("volume name already exists in project")
	ErrMaxVolumesExceeded  = errors.New("maximum number of volumes exceeded")
)

// Infrastructure errors - errors related to data persistence and external services
var (
	ErrDatabaseUnavailable    = errors.New("database unavailable")
	ErrDatabaseOperation      = errors.New("database operation failed")
	ErrConcurrentModification = errors.New("concurrent modification detected")
)

// Service errors - errors related to business service operations
var (
	ErrProjectCreationFailed   = errors.New("failed to create project")
	ErrOwnershipTransferFailed = errors.New("failed to transfer ownership")
	ErrSlugGenerationFailed    = errors.New("failed to generate unique slug")
)
