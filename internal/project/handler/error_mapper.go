package handler

import (
	"net/http"

	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// projectErrorMap provides mapping from project domain errors to response information
var projectErrorMap = map[error]response.ErrorMapping{
	// Repository errors
	projecterrors.ErrProjectNotFound:      {StatusCode: http.StatusNotFound, Code: "PROJECT_NOT_FOUND", Message: "Project not found"},
	projecterrors.ErrProjectAlreadyExists: {StatusCode: http.StatusConflict, Code: "PROJECT_ALREADY_EXISTS", Message: "Project already exists"},
	projecterrors.ErrProjectUserNotFound:  {StatusCode: http.StatusNotFound, Code: "PROJECT_USER_NOT_FOUND", Message: "Project user not found"},
	projecterrors.ErrSlugAlreadyExists:    {StatusCode: http.StatusConflict, Code: "SLUG_ALREADY_EXISTS", Message: "Slug already exists"},

	// Domain errors
	projecterrors.ErrInvalidProjectData:         {StatusCode: http.StatusBadRequest, Code: "INVALID_PROJECT_DATA", Message: "Invalid project data"},
	projecterrors.ErrProjectNotActive:           {StatusCode: http.StatusForbidden, Code: "PROJECT_NOT_ACTIVE", Message: "Project is not active"},
	projecterrors.ErrCannotModifyDeletedProject: {StatusCode: http.StatusForbidden, Code: "CANNOT_MODIFY_DELETED_PROJECT", Message: "Cannot modify deleted project"},

	// Permission errors
	projecterrors.ErrPermissionDenied:      {StatusCode: http.StatusForbidden, Code: "PERMISSION_DENIED", Message: "Permission denied"},
	projecterrors.ErrOwnerRequired:         {StatusCode: http.StatusForbidden, Code: "OWNER_REQUIRED", Message: "Owner permission required"},
	projecterrors.ErrCannotRemoveLastOwner: {StatusCode: http.StatusBadRequest, Code: "CANNOT_REMOVE_LAST_OWNER", Message: "Cannot remove last owner"},
	projecterrors.ErrUserAlreadyInProject:  {StatusCode: http.StatusConflict, Code: "USER_ALREADY_IN_PROJECT", Message: "User already in project"},
	projecterrors.ErrUserNotInProject:      {StatusCode: http.StatusNotFound, Code: "USER_NOT_IN_PROJECT", Message: "User not in project"},

	// Validation errors
	projecterrors.ErrNameRequired:     {StatusCode: http.StatusBadRequest, Code: "NAME_REQUIRED", Message: "Project name is required"},
	projecterrors.ErrSlugRequired:     {StatusCode: http.StatusBadRequest, Code: "SLUG_REQUIRED", Message: "Project slug is required"},
	projecterrors.ErrInvalidSlug:      {StatusCode: http.StatusBadRequest, Code: "INVALID_SLUG", Message: "Invalid project slug"},
	projecterrors.ErrInvalidProjectID: {StatusCode: http.StatusBadRequest, Code: "INVALID_PROJECT_ID", Message: "Invalid project ID"},
	projecterrors.ErrValidationFailed: {StatusCode: http.StatusBadRequest, Code: "VALIDATION_FAILED", Message: "Validation failed"},

	// Resource limit errors
	projecterrors.ErrResourceLimitExceeded: {StatusCode: http.StatusForbidden, Code: "RESOURCE_LIMIT_EXCEEDED", Message: "Resource limit exceeded"},
	projecterrors.ErrPlanLimitExceeded:     {StatusCode: http.StatusForbidden, Code: "PLAN_LIMIT_EXCEEDED", Message: "Plan limit exceeded"},

	// Volume errors
	projecterrors.ErrVolumeNotFound:      {StatusCode: http.StatusNotFound, Code: "VOLUME_NOT_FOUND", Message: "Volume not found"},
	projecterrors.ErrVolumeNameRequired:  {StatusCode: http.StatusBadRequest, Code: "VOLUME_NAME_REQUIRED", Message: "Volume name is required"},
	projecterrors.ErrInvalidCapacity:     {StatusCode: http.StatusBadRequest, Code: "INVALID_CAPACITY", Message: "Invalid volume capacity"},
	projecterrors.ErrDuplicateVolumeName: {StatusCode: http.StatusConflict, Code: "DUPLICATE_VOLUME_NAME", Message: "Volume name already exists"},

	// Infrastructure errors
	projecterrors.ErrDatabaseUnavailable: {StatusCode: http.StatusServiceUnavailable, Code: "DATABASE_UNAVAILABLE", Message: "Service temporarily unavailable"},
	projecterrors.ErrDatabaseOperation:   {StatusCode: http.StatusInternalServerError, Code: "DATABASE_OPERATION_FAILED", Message: "Database operation failed"},
}

// mapProjectError provides error mapping for project domain
func mapProjectError(err error) (response.ErrorMapping, bool) {
	if err == nil {
		return response.ErrorMapping{}, false
	}

	mapping, found := projectErrorMap[err]
	return mapping, found
}
