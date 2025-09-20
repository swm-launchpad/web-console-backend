package auth

// Error codes for auth package
const (
	// Authentication errors (AUTH_XXX)
	CodeInvalidCredentials    = "AUTH_001"
	CodeTokenExpired          = "AUTH_002"
	CodeInvalidToken          = "AUTH_003"
	CodeUnauthorized          = "AUTH_004"
	CodeInvalidRefreshToken   = "AUTH_005"
	CodeUserNotActive         = "AUTH_006"
	CodeTokenGenerationFailed = "AUTH_007"
	CodePasswordTooWeak       = "AUTH_008"
	CodePasswordMismatch      = "AUTH_009"
	CodeMissingAuthHeader     = "AUTH_010"
	CodeInvalidAuthFormat     = "AUTH_011"
	CodeMissingToken          = "AUTH_012"
)
