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
	projecterrors.ErrProjectNameExists:    {StatusCode: http.StatusConflict, Code: "PROJECT_NAME_EXISTS", Message: "Project name already exists for this user"},
	projecterrors.ErrProjectUserNotFound:  {StatusCode: http.StatusNotFound, Code: "PROJECT_USER_NOT_FOUND", Message: "Project user not found"},
	projecterrors.ErrSlugAlreadyExists:    {StatusCode: http.StatusConflict, Code: "SLUG_ALREADY_EXISTS", Message: "Slug already exists (globally unique)"},

	// Domain errors
	projecterrors.ErrInvalidProjectData:         {StatusCode: http.StatusBadRequest, Code: "INVALID_PROJECT_DATA", Message: "Invalid project data"},
	projecterrors.ErrProjectNotActive:           {StatusCode: http.StatusForbidden, Code: "PROJECT_NOT_ACTIVE", Message: "Project is not active"},
	projecterrors.ErrCannotModifyDeletedProject: {StatusCode: http.StatusForbidden, Code: "CANNOT_MODIFY_DELETED_PROJECT", Message: "Cannot modify deleted project"},
	projecterrors.ErrProjectAlreadyDeploying:    {StatusCode: http.StatusConflict, Code: "PROJECT_ALREADY_DEPLOYING", Message: "Project operation already in progress"},
	projecterrors.ErrProjectOperationInProgress: {StatusCode: http.StatusConflict, Code: "PROJECT_OPERATION_IN_PROGRESS", Message: "Cannot delete project: operation in progress (building or deploying)"},

	// Permission errors
	projecterrors.ErrPermissionDenied:      {StatusCode: http.StatusForbidden, Code: "PERMISSION_DENIED", Message: "Permission denied"},
	projecterrors.ErrOwnerRequired:         {StatusCode: http.StatusForbidden, Code: "OWNER_REQUIRED", Message: "Owner permission required"},
	projecterrors.ErrCannotRemoveLastOwner: {StatusCode: http.StatusBadRequest, Code: "CANNOT_REMOVE_LAST_OWNER", Message: "Cannot remove last owner"},
	projecterrors.ErrUserAlreadyInProject:  {StatusCode: http.StatusConflict, Code: "USER_ALREADY_IN_PROJECT", Message: "User already in project"},
	projecterrors.ErrUserNotInProject:      {StatusCode: http.StatusNotFound, Code: "USER_NOT_IN_PROJECT", Message: "User not in project"},

	// Validation errors
	projecterrors.ErrMissingField:      {StatusCode: http.StatusBadRequest, Code: "MISSING_FIELD", Message: "Required field is missing"},
	projecterrors.ErrNameRequired:      {StatusCode: http.StatusBadRequest, Code: "NAME_REQUIRED", Message: "Project name is required"},
	projecterrors.ErrSlugRequired:      {StatusCode: http.StatusBadRequest, Code: "SLUG_REQUIRED", Message: "Project slug is required"},
	projecterrors.ErrInvalidSlug:       {StatusCode: http.StatusBadRequest, Code: "INVALID_SLUG", Message: "Invalid project slug"},
	projecterrors.ErrSlugInvalidLength: {StatusCode: http.StatusBadRequest, Code: "SLUG_INVALID_LENGTH", Message: "Slug must be exactly 23 characters"},
	projecterrors.ErrSlugInvalidFormat: {StatusCode: http.StatusBadRequest, Code: "SLUG_INVALID_FORMAT", Message: "Slug has invalid format"},
	projecterrors.ErrInvalidProjectID:  {StatusCode: http.StatusBadRequest, Code: "INVALID_PROJECT_ID", Message: "Invalid project ID"},
	projecterrors.ErrValidationFailed:  {StatusCode: http.StatusBadRequest, Code: "VALIDATION_FAILED", Message: "Validation failed"},

	// Plan validation errors
	projecterrors.ErrInvalidPlan: {StatusCode: http.StatusBadRequest, Code: "INVALID_PLAN", Message: "Invalid plan type. Must be one of: free, eco, pro"},

	// Resource limit validation errors
	projecterrors.ErrCPULimitExceeded:     {StatusCode: http.StatusBadRequest, Code: "CPU_LIMIT_EXCEEDED", Message: "CPU limit must be between 500-8000 millicores (0.5-8 cores)"},
	projecterrors.ErrMemoryLimitExceeded:  {StatusCode: http.StatusBadRequest, Code: "MEMORY_LIMIT_EXCEEDED", Message: "Memory limit must be between 512Mi-16384Mi (0.5GB-16GB)"},
	projecterrors.ErrDiskLimitExceeded:    {StatusCode: http.StatusBadRequest, Code: "DISK_LIMIT_EXCEEDED", Message: "Disk limit must be between 1024Mi-3072Mi (1GB-3GB)"},
	projecterrors.ErrTrafficLimitExceeded: {StatusCode: http.StatusBadRequest, Code: "TRAFFIC_LIMIT_EXCEEDED", Message: "Traffic limit must be between 128Mi-1TB"},

	// Resource limit errors
	projecterrors.ErrResourceLimitExceeded:    {StatusCode: http.StatusForbidden, Code: "RESOURCE_LIMIT_EXCEEDED", Message: "Resource limit exceeded"},
	projecterrors.ErrPlanLimitExceeded:        {StatusCode: http.StatusForbidden, Code: "PLAN_LIMIT_EXCEEDED", Message: "Plan limit exceeded"},
	projecterrors.ErrProjectLimitExceeded:     {StatusCode: http.StatusBadRequest, Code: "PROJECT_LIMIT_EXCEEDED", Message: "Maximum number of projects exceeded"},
	projecterrors.ErrFreeResourcesFixed:       {StatusCode: http.StatusBadRequest, Code: "FREE_RESOURCES_FIXED", Message: "Free plan resources are fixed and cannot be modified (0.5 core, 1GB memory, 1GB disk)"},
	projecterrors.ErrFreeTierResourceExceeded: {StatusCode: http.StatusBadRequest, Code: "FREE_TIER_RESOURCE_EXCEEDED", Message: "Free tier resource limit exceeded: beta period allows up to 1 core CPU, 2GB memory, 3GB disk"},
	projecterrors.ErrFreePlanLimitExceeded:    {StatusCode: http.StatusBadRequest, Code: "FREE_PLAN_LIMIT_EXCEEDED", Message: "Free plan is limited to 1 project per user"},

	// Volume errors
	projecterrors.ErrVolumeNotFound:          {StatusCode: http.StatusNotFound, Code: "VOLUME_NOT_FOUND", Message: "Volume not found"},
	projecterrors.ErrVolumeNameRequired:      {StatusCode: http.StatusBadRequest, Code: "VOLUME_NAME_REQUIRED", Message: "Volume name is required"},
	projecterrors.ErrInvalidCapacity:         {StatusCode: http.StatusBadRequest, Code: "INVALID_CAPACITY", Message: "Invalid volume capacity"},
	projecterrors.ErrInvalidVolumeID:         {StatusCode: http.StatusBadRequest, Code: "INVALID_VOLUME_ID", Message: "Invalid volume ID"},
	projecterrors.ErrDuplicateVolumeName:     {StatusCode: http.StatusConflict, Code: "DUPLICATE_VOLUME_NAME", Message: "Volume name already exists"},
	projecterrors.ErrVolumeDiskLimitExceeded: {StatusCode: http.StatusBadRequest, Code: "VOLUME_DISK_LIMIT_EXCEEDED", Message: "Volume capacity exceeds project disk limit"},
	projecterrors.ErrVolumeCapacityTooSmall:  {StatusCode: http.StatusBadRequest, Code: "VOLUME_CAPACITY_TOO_SMALL", Message: "Volume capacity must be at least 128Mi"},
	projecterrors.ErrVolumeCapacityExceeded:  {StatusCode: http.StatusBadRequest, Code: "VOLUME_CAPACITY_EXCEEDED", Message: "Volume capacity exceeds maximum allowed (2048Mi)"},
	projecterrors.ErrInvalidVolumeName:       {StatusCode: http.StatusBadRequest, Code: "INVALID_VOLUME_NAME", Message: "Volume name must not exceed 255 characters"},
	projecterrors.ErrMaxVolumesExceeded:      {StatusCode: http.StatusBadRequest, Code: "MAX_VOLUMES_EXCEEDED", Message: "Maximum number of volumes exceeded"},

	// Infrastructure errors
	projecterrors.ErrDatabaseUnavailable:        {StatusCode: http.StatusServiceUnavailable, Code: "DATABASE_UNAVAILABLE", Message: "Service temporarily unavailable"},
	projecterrors.ErrDatabaseOperation:          {StatusCode: http.StatusInternalServerError, Code: "DATABASE_OPERATION_FAILED", Message: "Database operation failed"},
	projecterrors.ErrContainerConfigNotFound:    {StatusCode: http.StatusInternalServerError, Code: "CONTAINER_CONFIG_NOT_FOUND", Message: "Container configuration not found"},
	projecterrors.ErrKubernetesUnavailable:      {StatusCode: http.StatusServiceUnavailable, Code: "KUBERNETES_UNAVAILABLE", Message: "Kubernetes API unavailable"},
	projecterrors.ErrKubeAuthenticationFailed:   {StatusCode: http.StatusInternalServerError, Code: "KUBERNETES_AUTH_FAILED", Message: "Kubernetes authentication failed"},
	projecterrors.ErrKubeConnectionFailed:       {StatusCode: http.StatusServiceUnavailable, Code: "KUBERNETES_CONNECTION_FAILED", Message: "Kubernetes connection failed"},
	projecterrors.ErrKubeTimeout:                {StatusCode: http.StatusGatewayTimeout, Code: "KUBERNETES_TIMEOUT", Message: "Kubernetes API timeout"},
	projecterrors.ErrKubePipelineRunNotFound:    {StatusCode: http.StatusNotFound, Code: "PIPELINERUN_NOT_FOUND", Message: "PipelineRun not found"},
	projecterrors.ErrKubeUnknownError:           {StatusCode: http.StatusInternalServerError, Code: "KUBERNETES_ERROR", Message: "Kubernetes error"},
	projecterrors.ErrTektonUnavailable:          {StatusCode: http.StatusServiceUnavailable, Code: "TEKTON_UNAVAILABLE", Message: "Tekton API unavailable"},
	projecterrors.ErrTektonAuthenticationFailed: {StatusCode: http.StatusInternalServerError, Code: "TEKTON_AUTH_FAILED", Message: "Tekton authentication failed"},
	projecterrors.ErrInvalidDeploymentRequest:   {StatusCode: http.StatusBadRequest, Code: "INVALID_DEPLOYMENT_REQUEST", Message: "Invalid deployment request"},
	projecterrors.ErrTektonDeploymentFailed:     {StatusCode: http.StatusInternalServerError, Code: "DEPLOYMENT_TRIGGER_FAILED", Message: "Failed to trigger deployment"},
	projecterrors.ErrInvalidTektonResponse:      {StatusCode: http.StatusInternalServerError, Code: "INVALID_TEKTON_RESPONSE", Message: "Invalid Tekton response"},
	projecterrors.ErrNoRunningPods:              {StatusCode: http.StatusServiceUnavailable, Code: "NO_RUNNING_PODS", Message: "No application pods are currently running"},

	// Deployment errors
	projecterrors.ErrDeploymentNotFound:          {StatusCode: http.StatusNotFound, Code: "DEPLOYMENT_NOT_FOUND", Message: "Deployment not found"},
	projecterrors.ErrInvalidDeploymentStatus:     {StatusCode: http.StatusBadRequest, Code: "INVALID_DEPLOYMENT_STATUS", Message: "Invalid deployment status"},
	projecterrors.ErrCannotStartDeployment:       {StatusCode: http.StatusConflict, Code: "CANNOT_START_DEPLOYMENT", Message: "Cannot start deployment"},
	projecterrors.ErrCannotCompleteDeployment:    {StatusCode: http.StatusConflict, Code: "CANNOT_COMPLETE_DEPLOYMENT", Message: "Cannot complete deployment"},
	projecterrors.ErrCannotFailDeployment:        {StatusCode: http.StatusConflict, Code: "CANNOT_FAIL_DEPLOYMENT", Message: "Cannot fail deployment"},
	projecterrors.ErrCannotCancelDeployment:      {StatusCode: http.StatusConflict, Code: "CANNOT_CANCEL_DEPLOYMENT", Message: "Cannot cancel deployment"},
	projecterrors.ErrInvalidDeploymentTransition: {StatusCode: http.StatusConflict, Code: "INVALID_DEPLOYMENT_TRANSITION", Message: "Invalid deployment state transition"},
}

// mapProjectError provides error mapping for project domain
func mapProjectError(err error) (response.ErrorMapping, bool) {
	if err == nil {
		return response.ErrorMapping{}, false
	}

	mapping, found := projectErrorMap[err]
	return mapping, found
}
