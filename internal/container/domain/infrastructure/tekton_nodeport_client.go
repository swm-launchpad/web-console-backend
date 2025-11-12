package infrastructure

import (
	"context"
	"time"
)

// TektonNodePortClient defines the interface for managing temporary NodePort services
// through Tekton pipelines and Kubernetes API
type TektonNodePortClient interface {
	// TriggerNodePortCreation triggers the Tekton pipeline to create a temporary NodePort service
	// Returns the PipelineRun name immediately without waiting for completion
	TriggerNodePortCreation(ctx context.Context, params NodePortCreationParams) (string, error)

	// GetPipelineRunResult retrieves the result of a PipelineRun by its Tekton event ID
	// Returns NodePortInfo if the PipelineRun has completed successfully, nil if still running or not found
	GetPipelineRunResult(ctx context.Context, eventID string) (*NodePortInfo, error)

	// GetNodePortService retrieves the current status of a NodePort service
	// Returns nil if the service doesn't exist (expired or never created)
	GetNodePortService(ctx context.Context, serviceName string, namespace string) (*NodePortInfo, error)
}

// NodePortCreationParams contains parameters for creating a temporary NodePort
type NodePortCreationParams struct {
	ServiceName string // Knative Service name (project slug)
	Namespace   string // Kubernetes namespace (usually "application")
	TargetPort  int    // Container port to expose
	TTLSeconds  int    // Time-to-live in seconds
}

// NodePortInfo contains information about a NodePort service
type NodePortInfo struct {
	ServiceName string    // NodePort service name
	Namespace   string    // Kubernetes namespace
	Host        string    // DNS hostname (e.g., "r1.launchpad.kr")
	Port        int       // NodePort number (30000-32767)
	TargetPort  int       // Target container port
	Protocol    string    // Protocol (always "tcp")
	CreatedAt   time.Time // Service creation timestamp
	ExpiresAt   time.Time // Service expiration timestamp
	Status      string    // Service status: "active", "creating", "not_found"
}
