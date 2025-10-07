// Package dto defines Data Transfer Objects for infrastructure layer communications.
package dto

// TektonDeployResponse represents the response from Tekton EventListener after triggering a deployment.
// The EventListener returns this information to acknowledge that it received and processed the request.
//
// Note: This response does NOT include the actual PipelineRun name because Tekton uses Kubernetes
// generateName feature (deploy-run-xxxxx). To find the created PipelineRun, use KubeClient.ListPipelineRuns()
// with the project-id label.
type TektonDeployResponse struct {
	// EventListener is the name of the EventListener that processed the request
	// Example: "deploy-listener"
	EventListener string `json:"eventListener"`

	// EventListenerUID is the unique identifier of the EventListener resource
	// This is the Kubernetes UID of the EventListener
	EventListenerUID string `json:"eventListenerUID,omitempty"`

	// EventID is a unique identifier for this specific trigger event
	// Can be used for tracing and debugging
	EventID string `json:"eventID"`

	// Namespace is the Kubernetes namespace where the EventListener is running
	// Example: "deploy-pipeline"
	Namespace string `json:"namespace,omitempty"`
}
