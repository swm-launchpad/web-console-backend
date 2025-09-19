package auth

// Context keys for authentication-related values
const (
	// ContextKeyUserID is the context key for storing the authenticated user's ID
	ContextKeyUserID = "userID"

	// ContextKeyAuth is the context key for storing the authentication status
	ContextKeyAuth = "authenticated"
)