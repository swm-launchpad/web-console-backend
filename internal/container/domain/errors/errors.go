// Package errors defines domain-specific errors for the container bounded context.
// These errors represent business-level meanings without transport-specific codes.
package errors

import "errors"

// Repository errors - errors that occur at the data access layer
var (
	ErrContainerNotFound      = errors.New("container not found")
	ErrContainerAlreadyExists = errors.New("container already exists")
	ErrEnvVarNotFound         = errors.New("environment variable not found")
	ErrNetworkNotFound        = errors.New("network not found")
	ErrBuildHistoryNotFound   = errors.New("build history not found")
	ErrMountNotFound          = errors.New("mount not found")
)

// Domain errors - business logic violations and domain rule errors
var (
	ErrInvalidContainerData    = errors.New("invalid container data")
	ErrContainerNotActive      = errors.New("container is not active")
	ErrCannotModifyDeleted     = errors.New("cannot modify deleted container")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
)

// Permission errors - access control and authorization errors
var (
	ErrPermissionDenied      = errors.New("permission denied")
	ErrProjectAccessRequired = errors.New("project access required")
	ErrOwnerRequired         = errors.New("owner permission required")
	ErrAdminRequired         = errors.New("admin permission required")
)

// Validation errors - input validation and data integrity errors
var (
	ErrNameRequired      = errors.New("container name is required")
	ErrNameTooLong       = errors.New("container name must not exceed 32 characters")
	ErrSlugRequired      = errors.New("container slug is required")
	ErrInvalidSlug       = errors.New("invalid container slug")
	ErrSlugInvalidLength = errors.New("slug must be exactly 23 characters")
	// Deprecated: ErrSlugTooShort is unreachable with fixed 23-character slug format. Use ErrSlugInvalidLength instead.
	ErrSlugTooShort = errors.New("slug must be at least 3 characters long")
	// Deprecated: ErrSlugTooLong is unreachable with fixed 23-character slug format. Use ErrSlugInvalidLength instead.
	ErrSlugTooLong         = errors.New("slug must not exceed 63 characters")
	ErrSlugInvalidFormat   = errors.New("slug must follow format: c{timestamp}{random} (23 characters total)")
	ErrSlugReserved        = errors.New("slug is reserved")
	ErrSlugAlreadyExists   = errors.New("slug already exists (globally unique)")
	ErrContainerNameExists = errors.New("container name already exists in project")
	ErrInvalidProjectID    = errors.New("invalid project ID")
	ErrInvalidContainerID  = errors.New("invalid container ID")
	ErrValidationFailed    = errors.New("validation failed")
	ErrInvalidFormat       = errors.New("invalid format")
	ErrMissingField        = errors.New("required field is missing")
)

// Git errors - Git repository configuration errors
var (
	ErrGitURLRequired       = errors.New("git repository URL is required")
	ErrInvalidGitURL        = errors.New("invalid git repository URL")
	ErrInvalidGitBranch     = errors.New("invalid git branch")
	ErrInvalidGitPath       = errors.New("invalid git directory path")
	ErrInvalidGitCommitHash = errors.New("invalid git commit hash")
	ErrGitConfigRequired    = errors.New("git configuration is required")
)

// Environment variable errors
var (
	ErrEnvVarKeyRequired    = errors.New("environment variable key is required")
	ErrEnvVarValueRequired  = errors.New("environment variable value is required")
	ErrInvalidEnvVarKey     = errors.New("invalid environment variable key")
	ErrEnvVarKeyTooLong     = errors.New("environment variable key is too long")
	ErrEnvVarValueTooLong   = errors.New("environment variable value is too long")
	ErrDuplicateEnvVarKey   = errors.New("duplicate environment variable key")
	ErrReservedEnvVarKey    = errors.New("reserved environment variable key")
	ErrMaxEnvVarsExceeded   = errors.New("maximum number of environment variables exceeded")
	ErrCannotDeleteEnvVar   = errors.New("cannot delete environment variable")
	ErrEnvVarNotInContainer = errors.New("environment variable not in container")
)

// Secret errors - sensitive environment variable errors
var (
	ErrSecretNotFound       = errors.New("secret not found")
	ErrSecretKeyRequired    = errors.New("secret key is required")
	ErrSecretValueRequired  = errors.New("secret value is required")
	ErrInvalidSecretKey     = errors.New("invalid secret key")
	ErrSecretKeyTooLong     = errors.New("secret key is too long")
	ErrSecretValueTooLong   = errors.New("secret value is too long")
	ErrDuplicateSecretKey   = errors.New("duplicate secret key")
	ErrReservedSecretKey    = errors.New("reserved secret key")
	ErrMaxSecretsExceeded   = errors.New("maximum number of secrets exceeded")
	ErrCannotDeleteSecret   = errors.New("cannot delete secret")
	ErrSecretNotInContainer = errors.New("secret not in container")
)

// Build variable errors - build-time environment variable errors
var (
	ErrBuildVarNotFound       = errors.New("build variable not found")
	ErrBuildVarKeyRequired    = errors.New("build variable key is required")
	ErrBuildVarValueRequired  = errors.New("build variable value is required")
	ErrInvalidBuildVarKey     = errors.New("invalid build variable key")
	ErrBuildVarKeyTooLong     = errors.New("build variable key is too long")
	ErrBuildVarValueTooLong   = errors.New("build variable value is too long")
	ErrDuplicateBuildVarKey   = errors.New("duplicate build variable key")
	ErrReservedBuildVarKey    = errors.New("reserved build variable key")
	ErrMaxBuildVarsExceeded   = errors.New("maximum number of build variables exceeded")
	ErrCannotDeleteBuildVar   = errors.New("cannot delete build variable")
	ErrBuildVarNotInContainer = errors.New("build variable not in container")
)

// Cross-type key validation errors - keys must be unique across env_vars, secrets, and build_vars
var (
	ErrDuplicateKeyAcrossTypes = errors.New("key already exists in environment variables, secrets, or build variables")
)

// Network errors - network and port configuration errors
var (
	ErrInvalidPort           = errors.New("invalid port number")
	ErrInternalPortRequired  = errors.New("internal port is required")
	ErrExternalPortRequired  = errors.New("external port is required")
	ErrPortOutOfRange        = errors.New("port must be between 1 and 65535")
	ErrDuplicateInternalPort = errors.New("duplicate internal port in container")
	ErrExternalPortInUse     = errors.New("external port is already in use")
	ErrInvalidNetworkType    = errors.New("invalid network type")
	ErrMaxNetworksExceeded   = errors.New("maximum number of networks exceeded")
	ErrCannotDeleteNetwork   = errors.New("cannot delete network")
	ErrNetworkNotInContainer = errors.New("network not in container")
	ErrInvalidExternalIP     = errors.New("invalid external IP address")
	ErrExternalIPTooLong     = errors.New("external IP address is too long (max 45 characters)")
	ErrDuplicateHTTPNetwork  = errors.New("only one HTTP network allowed per container")
)

// Resource limit errors - errors related to resource constraints
var (
	ErrInvalidResourceLimits      = errors.New("invalid resource limits")
	ErrCPULimitNegative           = errors.New("CPU limit cannot be negative")
	ErrMemoryLimitNegative        = errors.New("memory limit cannot be negative")
	ErrCPULimitOutOfRange         = errors.New("CPU limit must be between 100-4000 millicores")
	ErrMemoryLimitOutOfRange      = errors.New("memory limit must be between 128-8192 Mi")
	ErrResourceLimitExceeded      = errors.New("resource limit exceeded")
	ErrInsufficientResources      = errors.New("insufficient resources")
	ErrProjectCPULimitExceeded    = errors.New("total container CPU usage would exceed project CPU limit")
	ErrProjectMemoryLimitExceeded = errors.New("total container memory usage would exceed project memory limit")
)

// Template errors - template configuration errors
var (
	ErrInvalidTemplateID     = errors.New("invalid template ID")
	ErrTemplateNotFound      = errors.New("template not found")
	ErrInvalidTemplateConfig = errors.New("invalid template configuration")
	ErrTemplateNotActive     = errors.New("template is not active")
)

// Mount errors - volume mount errors
var (
	ErrInvalidVolumeID      = errors.New("invalid volume ID")
	ErrVolumeNotFound       = errors.New("volume not found")
	ErrInvalidMountPath     = errors.New("invalid mount path")
	ErrMountPathRequired    = errors.New("mount path is required")
	ErrMountPathTooLong     = errors.New("mount path is too long")
	ErrDuplicateMountPath   = errors.New("duplicate mount path in container")
	ErrVolumeAlreadyMounted = errors.New("volume already mounted to container")
	ErrMaxMountsExceeded    = errors.New("maximum number of mounts exceeded")
	ErrCannotDeleteMount    = errors.New("cannot delete mount")
	ErrMountNotInContainer  = errors.New("mount not in container")
	ErrInvalidMountFormat   = errors.New("mount path must be absolute path")
	ErrMountPathReserved    = errors.New("mount path is reserved")
)

// Build errors - build history and process errors
var (
	ErrBuildInProgress    = errors.New("build already in progress")
	ErrInvalidBuildStatus = errors.New("invalid build status")
	ErrBuildFailed        = errors.New("build failed")
	ErrBuildCancelled     = errors.New("build cancelled")
	ErrInvalidTektonRef   = errors.New("invalid tekton reference")
)

// Infrastructure errors - errors related to data persistence and external services
var (
	ErrDatabaseUnavailable    = errors.New("database unavailable")
	ErrDatabaseOperation      = errors.New("database operation failed")
	ErrConcurrentModification = errors.New("concurrent modification detected")
	ErrTransactionFailed      = errors.New("transaction failed")
)

// Service errors - errors related to business service operations
var (
	ErrContainerCreationFailed = errors.New("failed to create container")
	ErrSlugGenerationFailed    = errors.New("failed to generate unique slug")
	ErrContainerUpdateFailed   = errors.New("failed to update container")
	ErrContainerDeletionFailed = errors.New("failed to delete container")
)

// Limit errors - project and container limits
var (
	ErrMaxContainersPerProject = errors.New("maximum number of containers per project exceeded")
	ErrProjectNotFound         = errors.New("project not found")
	ErrProjectInactive         = errors.New("project is not active")
)

// State errors - container state related errors
var (
	ErrCannotStartContainer   = errors.New("cannot start container")
	ErrCannotStopContainer    = errors.New("cannot stop container")
	ErrCannotRestartContainer = errors.New("cannot restart container")
	ErrCannotResetContainer   = errors.New("cannot reset container")
	ErrInvalidContainerState  = errors.New("invalid container state")
)

// FQDN and domain errors
var (
	ErrInvalidFQDN   = errors.New("invalid fully qualified domain name")
	ErrFQDNTooLong   = errors.New("FQDN is too long")
	ErrFQDNTooShort  = errors.New("subdomain must be at least 4 characters")
	ErrDuplicateFQDN = errors.New("FQDN already exists")
	ErrReservedFQDN  = errors.New("subdomain is reserved for system use")
)

// Stable window errors
var (
	ErrInvalidStableWindow = errors.New("invalid stable window")
	ErrStableWindowTooLong = errors.New("stable window is too long")
)

// Monthly metrics errors
var (
	ErrInvalidMonthlyMetrics = errors.New("invalid monthly metrics")
	ErrNegativeBuildTime     = errors.New("build time cannot be negative")
	ErrNegativeBuildCount    = errors.New("build count cannot be negative")
	ErrInvalidUptime         = errors.New("uptime must be between 0 and 100")
)

// NodePort errors - temporary NodePort service errors
var (
	ErrNodePortNotSupported       = errors.New("nodeport only supports TCP networks")
	ErrNodePortAlreadyExists      = errors.New("nodeport already exists for this container")
	ErrNodePortNotFound           = errors.New("nodeport service not found")
	ErrNodePortCreating           = errors.New("nodeport is being created")
	ErrNodePortAlreadyActive      = errors.New("nodeport is already active")
	ErrTektonUnavailable          = errors.New("tekton service unavailable")
	ErrTektonPipelineFailed       = errors.New("tekton pipeline execution failed")
	ErrTektonPipelineTimeout      = errors.New("tekton pipeline execution timeout")
	ErrKubernetesUnavailable      = errors.New("kubernetes service unavailable")
	ErrInvalidRequest             = errors.New("invalid request")
	ErrNoTCPNetwork               = errors.New("container has no TCP network configured")
	ErrMultipleNetworksNotAllowed = errors.New("container has multiple networks, cannot determine target port")
)

// Webhook errors - webhook token and auto-deployment errors
var (
	ErrInvalidWebhookToken       = errors.New("invalid webhook token format")
	ErrWebhookTokenNotFound      = errors.New("webhook token not found")
	ErrDuplicateWebhookToken     = errors.New("webhook token already exists")
	ErrContainerHasNoToken       = errors.New("container does not have webhook token")
	ErrContainerAlreadyHasToken  = errors.New("container already has webhook token")
	ErrWebhookNotEnabled         = errors.New("webhook is not enabled for this container")
	ErrWebhookAlreadyEnabled     = errors.New("webhook is already enabled for this container")
	ErrUnauthorizedWebhookAccess = errors.New("unauthorized access to webhook")
)
