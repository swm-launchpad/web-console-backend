package deployment

// DefaultNamespace is the Kubernetes namespace where all deployments are executed.
// This is a fixed value across the entire application.
const DefaultNamespace = "application"

// DefaultStableWindow is the default observation period in seconds for Knative's
// scale-to-zero decision making. This determines how long the system waits before
// scaling down idle services.
const DefaultStableWindow = 180
