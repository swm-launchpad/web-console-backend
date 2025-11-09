package handler

import (
	"errors"
	"net/http"

	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// containerErrorMap provides mapping from container domain errors to response information
var containerErrorMap = map[error]response.ErrorMapping{
	// Repository errors
	containererrors.ErrContainerNotFound:      {StatusCode: http.StatusNotFound, Code: "CONTAINER_NOT_FOUND", Message: "Container not found"},
	containererrors.ErrContainerAlreadyExists: {StatusCode: http.StatusConflict, Code: "CONTAINER_ALREADY_EXISTS", Message: "Container already exists"},
	containererrors.ErrEnvVarNotFound:         {StatusCode: http.StatusNotFound, Code: "ENV_VAR_NOT_FOUND", Message: "Environment variable not found"},
	containererrors.ErrNetworkNotFound:        {StatusCode: http.StatusNotFound, Code: "NETWORK_NOT_FOUND", Message: "Network not found"},
	containererrors.ErrBuildHistoryNotFound:   {StatusCode: http.StatusNotFound, Code: "BUILD_HISTORY_NOT_FOUND", Message: "Build history not found"},
	projecterrors.ErrBuildHistoryNotFound:     {StatusCode: http.StatusNotFound, Code: "BUILD_HISTORY_NOT_FOUND", Message: "Build history not found"},
	containererrors.ErrMountNotFound:          {StatusCode: http.StatusNotFound, Code: "MOUNT_NOT_FOUND", Message: "Mount not found"},

	// Domain errors
	containererrors.ErrInvalidContainerData:    {StatusCode: http.StatusBadRequest, Code: "INVALID_CONTAINER_DATA", Message: "Invalid container data"},
	containererrors.ErrContainerNotActive:      {StatusCode: http.StatusBadRequest, Code: "CONTAINER_NOT_ACTIVE", Message: "Container is not active"},
	containererrors.ErrCannotModifyDeleted:     {StatusCode: http.StatusBadRequest, Code: "CANNOT_MODIFY_DELETED", Message: "Cannot modify deleted container"},
	containererrors.ErrInvalidStatusTransition: {StatusCode: http.StatusBadRequest, Code: "INVALID_STATUS_TRANSITION", Message: "Invalid status transition"},

	// Permission errors
	containererrors.ErrPermissionDenied:      {StatusCode: http.StatusForbidden, Code: "PERMISSION_DENIED", Message: "Permission denied"},
	containererrors.ErrProjectAccessRequired: {StatusCode: http.StatusForbidden, Code: "PROJECT_ACCESS_REQUIRED", Message: "Project access required"},
	containererrors.ErrOwnerRequired:         {StatusCode: http.StatusForbidden, Code: "OWNER_REQUIRED", Message: "Owner permission required"},
	containererrors.ErrAdminRequired:         {StatusCode: http.StatusForbidden, Code: "ADMIN_REQUIRED", Message: "Admin permission required"},

	// Validation errors
	containererrors.ErrNameRequired:        {StatusCode: http.StatusBadRequest, Code: "NAME_REQUIRED", Message: "Container name is required"},
	containererrors.ErrNameTooLong:         {StatusCode: http.StatusBadRequest, Code: "NAME_TOO_LONG", Message: "Container name must not exceed 32 characters"},
	containererrors.ErrSlugRequired:        {StatusCode: http.StatusBadRequest, Code: "SLUG_REQUIRED", Message: "Container slug is required"},
	containererrors.ErrInvalidSlug:         {StatusCode: http.StatusBadRequest, Code: "INVALID_SLUG", Message: "Invalid container slug"},
	containererrors.ErrSlugInvalidFormat:   {StatusCode: http.StatusBadRequest, Code: "SLUG_INVALID_FORMAT", Message: "Slug can only contain lowercase letters, numbers, and hyphens"},
	containererrors.ErrSlugReserved:        {StatusCode: http.StatusBadRequest, Code: "SLUG_RESERVED", Message: "Slug is reserved"},
	containererrors.ErrSlugAlreadyExists:   {StatusCode: http.StatusConflict, Code: "SLUG_ALREADY_EXISTS", Message: "Slug already exists (globally unique)"},
	containererrors.ErrContainerNameExists: {StatusCode: http.StatusConflict, Code: "CONTAINER_NAME_EXISTS", Message: "Container name already exists in project"},
	containererrors.ErrInvalidProjectID:    {StatusCode: http.StatusBadRequest, Code: "INVALID_PROJECT_ID", Message: "Invalid project ID"},
	containererrors.ErrInvalidContainerID:  {StatusCode: http.StatusBadRequest, Code: "INVALID_CONTAINER_ID", Message: "Invalid container ID"},
	containererrors.ErrValidationFailed:    {StatusCode: http.StatusBadRequest, Code: "VALIDATION_FAILED", Message: "Validation failed"},
	containererrors.ErrInvalidFormat:       {StatusCode: http.StatusBadRequest, Code: "INVALID_FORMAT", Message: "Invalid format"},
	containererrors.ErrMissingField:        {StatusCode: http.StatusBadRequest, Code: "MISSING_FIELD", Message: "Required field is missing"},

	// Git errors
	containererrors.ErrGitURLRequired:       {StatusCode: http.StatusBadRequest, Code: "GIT_URL_REQUIRED", Message: "Git repository URL is required"},
	containererrors.ErrInvalidGitURL:        {StatusCode: http.StatusBadRequest, Code: "INVALID_GIT_URL", Message: "Invalid git repository URL"},
	containererrors.ErrInvalidGitBranch:     {StatusCode: http.StatusBadRequest, Code: "INVALID_GIT_BRANCH", Message: "Invalid git branch"},
	containererrors.ErrInvalidGitPath:       {StatusCode: http.StatusBadRequest, Code: "INVALID_GIT_PATH", Message: "Invalid git directory path"},
	containererrors.ErrInvalidGitCommitHash: {StatusCode: http.StatusBadRequest, Code: "INVALID_GIT_COMMIT_HASH", Message: "Invalid git commit hash"},
	containererrors.ErrGitConfigRequired:    {StatusCode: http.StatusBadRequest, Code: "GIT_CONFIG_REQUIRED", Message: "Git configuration is required"},

	// Environment variable errors
	containererrors.ErrEnvVarKeyRequired:    {StatusCode: http.StatusBadRequest, Code: "ENV_VAR_KEY_REQUIRED", Message: "Environment variable key is required"},
	containererrors.ErrEnvVarValueRequired:  {StatusCode: http.StatusBadRequest, Code: "ENV_VAR_VALUE_REQUIRED", Message: "Environment variable value is required"},
	containererrors.ErrInvalidEnvVarKey:     {StatusCode: http.StatusBadRequest, Code: "INVALID_ENV_VAR_KEY", Message: "Invalid environment variable key"},
	containererrors.ErrEnvVarKeyTooLong:     {StatusCode: http.StatusBadRequest, Code: "ENV_VAR_KEY_TOO_LONG", Message: "Environment variable key is too long"},
	containererrors.ErrEnvVarValueTooLong:   {StatusCode: http.StatusBadRequest, Code: "ENV_VAR_VALUE_TOO_LONG", Message: "Environment variable value is too long"},
	containererrors.ErrDuplicateEnvVarKey:   {StatusCode: http.StatusConflict, Code: "DUPLICATE_ENV_VAR_KEY", Message: "Duplicate environment variable key"},
	containererrors.ErrReservedEnvVarKey:    {StatusCode: http.StatusBadRequest, Code: "RESERVED_ENV_VAR_KEY", Message: "Reserved environment variable key"},
	containererrors.ErrMaxEnvVarsExceeded:   {StatusCode: http.StatusBadRequest, Code: "MAX_ENV_VARS_EXCEEDED", Message: "Maximum number of environment variables exceeded"},
	containererrors.ErrCannotDeleteEnvVar:   {StatusCode: http.StatusBadRequest, Code: "CANNOT_DELETE_ENV_VAR", Message: "Cannot delete environment variable"},
	containererrors.ErrEnvVarNotInContainer: {StatusCode: http.StatusBadRequest, Code: "ENV_VAR_NOT_IN_CONTAINER", Message: "Environment variable not in container"},

	// Secret errors
	containererrors.ErrSecretNotFound:       {StatusCode: http.StatusNotFound, Code: "SECRET_NOT_FOUND", Message: "Secret not found"},
	containererrors.ErrSecretKeyRequired:    {StatusCode: http.StatusBadRequest, Code: "SECRET_KEY_REQUIRED", Message: "Secret key is required"},
	containererrors.ErrSecretValueRequired:  {StatusCode: http.StatusBadRequest, Code: "SECRET_VALUE_REQUIRED", Message: "Secret value is required"},
	containererrors.ErrInvalidSecretKey:     {StatusCode: http.StatusBadRequest, Code: "INVALID_SECRET_KEY", Message: "Invalid secret key"},
	containererrors.ErrSecretKeyTooLong:     {StatusCode: http.StatusBadRequest, Code: "SECRET_KEY_TOO_LONG", Message: "Secret key is too long"},
	containererrors.ErrSecretValueTooLong:   {StatusCode: http.StatusBadRequest, Code: "SECRET_VALUE_TOO_LONG", Message: "Secret value is too long"},
	containererrors.ErrDuplicateSecretKey:   {StatusCode: http.StatusConflict, Code: "DUPLICATE_SECRET_KEY", Message: "Duplicate secret key"},
	containererrors.ErrReservedSecretKey:    {StatusCode: http.StatusBadRequest, Code: "RESERVED_SECRET_KEY", Message: "Reserved secret key"},
	containererrors.ErrMaxSecretsExceeded:   {StatusCode: http.StatusBadRequest, Code: "MAX_SECRETS_EXCEEDED", Message: "Maximum number of secrets exceeded"},
	containererrors.ErrCannotDeleteSecret:   {StatusCode: http.StatusBadRequest, Code: "CANNOT_DELETE_SECRET", Message: "Cannot delete secret"},
	containererrors.ErrSecretNotInContainer: {StatusCode: http.StatusBadRequest, Code: "SECRET_NOT_IN_CONTAINER", Message: "Secret not in container"},

	// Build variable errors
	containererrors.ErrBuildVarNotFound:       {StatusCode: http.StatusNotFound, Code: "BUILD_VAR_NOT_FOUND", Message: "Build variable not found"},
	containererrors.ErrBuildVarKeyRequired:    {StatusCode: http.StatusBadRequest, Code: "BUILD_VAR_KEY_REQUIRED", Message: "Build variable key is required"},
	containererrors.ErrBuildVarValueRequired:  {StatusCode: http.StatusBadRequest, Code: "BUILD_VAR_VALUE_REQUIRED", Message: "Build variable value is required"},
	containererrors.ErrInvalidBuildVarKey:     {StatusCode: http.StatusBadRequest, Code: "INVALID_BUILD_VAR_KEY", Message: "Invalid build variable key"},
	containererrors.ErrBuildVarKeyTooLong:     {StatusCode: http.StatusBadRequest, Code: "BUILD_VAR_KEY_TOO_LONG", Message: "Build variable key is too long"},
	containererrors.ErrBuildVarValueTooLong:   {StatusCode: http.StatusBadRequest, Code: "BUILD_VAR_VALUE_TOO_LONG", Message: "Build variable value is too long"},
	containererrors.ErrDuplicateBuildVarKey:   {StatusCode: http.StatusConflict, Code: "DUPLICATE_BUILD_VAR_KEY", Message: "Duplicate build variable key"},
	containererrors.ErrReservedBuildVarKey:    {StatusCode: http.StatusBadRequest, Code: "RESERVED_BUILD_VAR_KEY", Message: "Reserved build variable key"},
	containererrors.ErrMaxBuildVarsExceeded:   {StatusCode: http.StatusBadRequest, Code: "MAX_BUILD_VARS_EXCEEDED", Message: "Maximum number of build variables exceeded"},
	containererrors.ErrCannotDeleteBuildVar:   {StatusCode: http.StatusBadRequest, Code: "CANNOT_DELETE_BUILD_VAR", Message: "Cannot delete build variable"},
	containererrors.ErrBuildVarNotInContainer: {StatusCode: http.StatusBadRequest, Code: "BUILD_VAR_NOT_IN_CONTAINER", Message: "Build variable not in container"},

	// Cross-type key validation errors
	containererrors.ErrDuplicateKeyAcrossTypes: {StatusCode: http.StatusConflict, Code: "DUPLICATE_KEY_ACROSS_TYPES", Message: "Key already exists in environment variables, secrets, or build variables"},

	// Network errors
	containererrors.ErrInvalidPort:           {StatusCode: http.StatusBadRequest, Code: "INVALID_PORT", Message: "Invalid port number"},
	containererrors.ErrInternalPortRequired:  {StatusCode: http.StatusBadRequest, Code: "INTERNAL_PORT_REQUIRED", Message: "Internal port is required"},
	containererrors.ErrExternalPortRequired:  {StatusCode: http.StatusBadRequest, Code: "EXTERNAL_PORT_REQUIRED", Message: "External port is required"},
	containererrors.ErrPortOutOfRange:        {StatusCode: http.StatusBadRequest, Code: "PORT_OUT_OF_RANGE", Message: "Port must be between 1 and 65535"},
	containererrors.ErrDuplicateInternalPort: {StatusCode: http.StatusConflict, Code: "DUPLICATE_INTERNAL_PORT", Message: "Duplicate internal port in container"},
	containererrors.ErrExternalPortInUse:     {StatusCode: http.StatusConflict, Code: "EXTERNAL_PORT_IN_USE", Message: "External port is already in use"},
	containererrors.ErrInvalidNetworkType:    {StatusCode: http.StatusBadRequest, Code: "INVALID_NETWORK_TYPE", Message: "Invalid network type"},
	containererrors.ErrMaxNetworksExceeded:   {StatusCode: http.StatusBadRequest, Code: "MAX_NETWORKS_EXCEEDED", Message: "Maximum number of networks exceeded"},
	containererrors.ErrCannotDeleteNetwork:   {StatusCode: http.StatusBadRequest, Code: "CANNOT_DELETE_NETWORK", Message: "Cannot delete network"},
	containererrors.ErrNetworkNotInContainer: {StatusCode: http.StatusBadRequest, Code: "NETWORK_NOT_IN_CONTAINER", Message: "Network not in container"},
	containererrors.ErrInvalidExternalIP:     {StatusCode: http.StatusBadRequest, Code: "INVALID_EXTERNAL_IP", Message: "Invalid external IP address"},
	containererrors.ErrExternalIPTooLong:     {StatusCode: http.StatusBadRequest, Code: "EXTERNAL_IP_TOO_LONG", Message: "External IP address is too long (max 45 characters)"},

	// Resource limit errors
	containererrors.ErrInvalidResourceLimits:      {StatusCode: http.StatusBadRequest, Code: "INVALID_RESOURCE_LIMITS", Message: "Invalid resource limits"},
	containererrors.ErrCPULimitNegative:           {StatusCode: http.StatusBadRequest, Code: "CPU_LIMIT_NEGATIVE", Message: "CPU limit cannot be negative"},
	containererrors.ErrMemoryLimitNegative:        {StatusCode: http.StatusBadRequest, Code: "MEMORY_LIMIT_NEGATIVE", Message: "Memory limit cannot be negative"},
	containererrors.ErrCPULimitOutOfRange:         {StatusCode: http.StatusBadRequest, Code: "CPU_LIMIT_OUT_OF_RANGE", Message: "CPU limit must be between 100-4000 millicores"},
	containererrors.ErrMemoryLimitOutOfRange:      {StatusCode: http.StatusBadRequest, Code: "MEMORY_LIMIT_OUT_OF_RANGE", Message: "Memory limit must be between 128-8192 Mi"},
	containererrors.ErrResourceLimitExceeded:      {StatusCode: http.StatusBadRequest, Code: "RESOURCE_LIMIT_EXCEEDED", Message: "Resource limit exceeded"},
	containererrors.ErrInsufficientResources:      {StatusCode: http.StatusServiceUnavailable, Code: "INSUFFICIENT_RESOURCES", Message: "Insufficient resources"},
	containererrors.ErrProjectCPULimitExceeded:    {StatusCode: http.StatusUnprocessableEntity, Code: "PROJECT_CPU_LIMIT_EXCEEDED", Message: "Total container CPU usage would exceed project CPU limit"},
	containererrors.ErrProjectMemoryLimitExceeded: {StatusCode: http.StatusUnprocessableEntity, Code: "PROJECT_MEMORY_LIMIT_EXCEEDED", Message: "Total container memory usage would exceed project memory limit"},

	// Template errors
	containererrors.ErrInvalidTemplateID:     {StatusCode: http.StatusBadRequest, Code: "INVALID_TEMPLATE_ID", Message: "Invalid template ID"},
	containererrors.ErrTemplateNotFound:      {StatusCode: http.StatusNotFound, Code: "TEMPLATE_NOT_FOUND", Message: "Template not found"},
	containererrors.ErrInvalidTemplateConfig: {StatusCode: http.StatusBadRequest, Code: "INVALID_TEMPLATE_CONFIG", Message: "Invalid template configuration"},
	containererrors.ErrTemplateNotActive:     {StatusCode: http.StatusBadRequest, Code: "TEMPLATE_NOT_ACTIVE", Message: "Template is not active"},

	// Mount errors
	containererrors.ErrInvalidVolumeID:      {StatusCode: http.StatusBadRequest, Code: "INVALID_VOLUME_ID", Message: "Invalid volume ID"},
	containererrors.ErrVolumeNotFound:       {StatusCode: http.StatusNotFound, Code: "VOLUME_NOT_FOUND", Message: "Volume not found"},
	containererrors.ErrInvalidMountPath:     {StatusCode: http.StatusBadRequest, Code: "INVALID_MOUNT_PATH", Message: "Invalid mount path"},
	containererrors.ErrMountPathRequired:    {StatusCode: http.StatusBadRequest, Code: "MOUNT_PATH_REQUIRED", Message: "Mount path is required"},
	containererrors.ErrMountPathTooLong:     {StatusCode: http.StatusBadRequest, Code: "MOUNT_PATH_TOO_LONG", Message: "Mount path is too long"},
	containererrors.ErrDuplicateMountPath:   {StatusCode: http.StatusConflict, Code: "DUPLICATE_MOUNT_PATH", Message: "Duplicate mount path in container"},
	containererrors.ErrVolumeAlreadyMounted: {StatusCode: http.StatusConflict, Code: "VOLUME_ALREADY_MOUNTED", Message: "Volume already mounted to container"},
	containererrors.ErrMaxMountsExceeded:    {StatusCode: http.StatusBadRequest, Code: "MAX_MOUNTS_EXCEEDED", Message: "Maximum number of mounts exceeded"},
	containererrors.ErrCannotDeleteMount:    {StatusCode: http.StatusBadRequest, Code: "CANNOT_DELETE_MOUNT", Message: "Cannot delete mount"},
	containererrors.ErrMountNotInContainer:  {StatusCode: http.StatusBadRequest, Code: "MOUNT_NOT_IN_CONTAINER", Message: "Mount not in container"},
	containererrors.ErrInvalidMountFormat:   {StatusCode: http.StatusBadRequest, Code: "INVALID_MOUNT_FORMAT", Message: "Mount path must be absolute path"},
	containererrors.ErrMountPathReserved:    {StatusCode: http.StatusBadRequest, Code: "MOUNT_PATH_RESERVED", Message: "Mount path is reserved"},

	// Build errors
	containererrors.ErrBuildInProgress:    {StatusCode: http.StatusConflict, Code: "BUILD_IN_PROGRESS", Message: "Build already in progress"},
	containererrors.ErrInvalidBuildStatus: {StatusCode: http.StatusBadRequest, Code: "INVALID_BUILD_STATUS", Message: "Invalid build status"},
	containererrors.ErrBuildFailed:        {StatusCode: http.StatusInternalServerError, Code: "BUILD_FAILED", Message: "Build failed"},
	containererrors.ErrBuildCancelled:     {StatusCode: http.StatusBadRequest, Code: "BUILD_CANCELLED", Message: "Build cancelled"},
	containererrors.ErrInvalidTektonRef:   {StatusCode: http.StatusBadRequest, Code: "INVALID_TEKTON_REF", Message: "Invalid tekton reference"},

	// Infrastructure errors
	containererrors.ErrDatabaseUnavailable:    {StatusCode: http.StatusServiceUnavailable, Code: "DATABASE_UNAVAILABLE", Message: "Database unavailable"},
	containererrors.ErrDatabaseOperation:      {StatusCode: http.StatusInternalServerError, Code: "DATABASE_OPERATION_FAILED", Message: "Database operation failed"},
	containererrors.ErrConcurrentModification: {StatusCode: http.StatusConflict, Code: "CONCURRENT_MODIFICATION", Message: "Concurrent modification detected"},
	containererrors.ErrTransactionFailed:      {StatusCode: http.StatusInternalServerError, Code: "TRANSACTION_FAILED", Message: "Transaction failed"},

	// Service errors
	containererrors.ErrContainerCreationFailed: {StatusCode: http.StatusInternalServerError, Code: "CONTAINER_CREATION_FAILED", Message: "Failed to create container"},
	containererrors.ErrSlugGenerationFailed:    {StatusCode: http.StatusInternalServerError, Code: "SLUG_GENERATION_FAILED", Message: "Failed to generate unique slug"},
	containererrors.ErrContainerUpdateFailed:   {StatusCode: http.StatusInternalServerError, Code: "CONTAINER_UPDATE_FAILED", Message: "Failed to update container"},
	containererrors.ErrContainerDeletionFailed: {StatusCode: http.StatusInternalServerError, Code: "CONTAINER_DELETION_FAILED", Message: "Failed to delete container"},

	// Limit errors
	containererrors.ErrMaxContainersPerProject: {StatusCode: http.StatusBadRequest, Code: "MAX_CONTAINERS_PER_PROJECT", Message: "Maximum number of containers per project exceeded"},
	containererrors.ErrProjectNotFound:         {StatusCode: http.StatusNotFound, Code: "PROJECT_NOT_FOUND", Message: "Project not found"},
	containererrors.ErrProjectInactive:         {StatusCode: http.StatusBadRequest, Code: "PROJECT_INACTIVE", Message: "Project is not active"},

	// State errors
	containererrors.ErrCannotStartContainer:   {StatusCode: http.StatusBadRequest, Code: "CANNOT_START_CONTAINER", Message: "Cannot start container"},
	containererrors.ErrCannotStopContainer:    {StatusCode: http.StatusBadRequest, Code: "CANNOT_STOP_CONTAINER", Message: "Cannot stop container"},
	containererrors.ErrCannotRestartContainer: {StatusCode: http.StatusBadRequest, Code: "CANNOT_RESTART_CONTAINER", Message: "Cannot restart container"},
	containererrors.ErrCannotResetContainer:   {StatusCode: http.StatusBadRequest, Code: "CANNOT_RESET_CONTAINER", Message: "Cannot reset container"},
	containererrors.ErrInvalidContainerState:  {StatusCode: http.StatusBadRequest, Code: "INVALID_CONTAINER_STATE", Message: "Invalid container state"},

	// FQDN and domain errors
	containererrors.ErrInvalidFQDN:   {StatusCode: http.StatusBadRequest, Code: "INVALID_FQDN", Message: "Invalid fully qualified domain name"},
	containererrors.ErrFQDNTooLong:   {StatusCode: http.StatusBadRequest, Code: "FQDN_TOO_LONG", Message: "FQDN is too long"},
	containererrors.ErrFQDNTooShort:  {StatusCode: http.StatusBadRequest, Code: "FQDN_TOO_SHORT", Message: "Subdomain must be at least 4 characters"},
	containererrors.ErrDuplicateFQDN: {StatusCode: http.StatusConflict, Code: "DUPLICATE_FQDN", Message: "FQDN already exists"},
	containererrors.ErrReservedFQDN:  {StatusCode: http.StatusBadRequest, Code: "RESERVED_FQDN", Message: "Subdomain is reserved for system use"},

	// Stable window errors
	containererrors.ErrInvalidStableWindow: {StatusCode: http.StatusBadRequest, Code: "INVALID_STABLE_WINDOW", Message: "Invalid stable window"},
	containererrors.ErrStableWindowTooLong: {StatusCode: http.StatusBadRequest, Code: "STABLE_WINDOW_TOO_LONG", Message: "Stable window is too long"},

	// Monthly metrics errors
	containererrors.ErrInvalidMonthlyMetrics: {StatusCode: http.StatusBadRequest, Code: "INVALID_MONTHLY_METRICS", Message: "Invalid monthly metrics"},
	containererrors.ErrNegativeBuildTime:     {StatusCode: http.StatusBadRequest, Code: "NEGATIVE_BUILD_TIME", Message: "Build time cannot be negative"},
	containererrors.ErrNegativeBuildCount:    {StatusCode: http.StatusBadRequest, Code: "NEGATIVE_BUILD_COUNT", Message: "Build count cannot be negative"},
	containererrors.ErrInvalidUptime:         {StatusCode: http.StatusBadRequest, Code: "INVALID_UPTIME", Message: "Uptime must be between 0 and 100"},
}

// mapContainerError provides error mapping for container domain
// Uses errors.Is to support wrapped errors
func mapContainerError(err error) (response.ErrorMapping, bool) {
	if err == nil {
		return response.ErrorMapping{}, false
	}

	// Iterate through error map and use errors.Is for comparison
	// This supports wrapped errors (e.g., fmt.Errorf("context: %w", err))
	for domainErr, mapping := range containerErrorMap {
		if errors.Is(err, domainErr) {
			return mapping, true
		}
	}

	return response.ErrorMapping{}, false
}
