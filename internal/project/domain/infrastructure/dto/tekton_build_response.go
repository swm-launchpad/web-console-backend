// Package dto defines Data Transfer Objects for infrastructure layer communications.
package dto

// TektonBuildResponse represents the response from Tekton EventListener after triggering a build.
// The EventListener returns this information to acknowledge that it received and processed the request.
//
// Note: This response does NOT include the actual PipelineRun name because Tekton uses Kubernetes
// generateName feature (image-build-push-run-xxxxx). To find the created PipelineRun, use
// KubeBuildClient.FindPipelineRunNameByEventID() with the EventID from this response.
//
// The EventID can be used to:
//   - Track the build in logs and monitoring systems
//   - Query the PipelineRun status using Kubernetes label selector
//   - Correlate backend build records with Kubernetes PipelineRuns
type TektonBuildResponse struct {
	// EventListener is the name of the EventListener that processed the request
	// Example: "image-build-push"
	EventListener string `json:"eventListener"`

	// EventListenerUID is the unique identifier of the EventListener resource
	// This is the Kubernetes UID of the EventListener
	EventListenerUID string `json:"eventListenerUID,omitempty"`

	// EventID is a unique identifier for this specific trigger event
	// Can be used for tracing, debugging, and finding the associated PipelineRun
	// The PipelineRun will have a label: triggers.tekton.dev/triggers-eventid=<EventID>
	EventID string `json:"eventID"`

	// Namespace is the Kubernetes namespace where the EventListener is running
	// Example: "build-pipeline"
	Namespace string `json:"namespace,omitempty"`
}
