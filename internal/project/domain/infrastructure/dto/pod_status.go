// Package dto defines Data Transfer Objects for infrastructure layer communications.
package dto

// PodStatus represents the status of a Kubernetes Pod for a project.
// This is used to check if application pods are running and healthy.
type PodStatus struct {
	// Exists indicates whether at least one pod exists for the project
	Exists bool

	// Status is the kubectl-style status string (e.g., "Running", "ContainerCreating", "CrashLoopBackOff")
	// This matches what kubectl shows in the STATUS column
	// Empty if no pod exists
	Status string

	// Phase is the raw Kubernetes pod phase (e.g., "Running", "Pending", "Failed")
	// Empty if no pod exists
	Phase string

	// Reason provides additional context for the status (optional)
	// Examples: "ContainerCreating", "CrashLoopBackOff", "ImagePullBackOff"
	Reason string

	// ReadyContainers is the number of containers that are ready
	// 0 if no pod exists
	ReadyContainers int

	// TotalContainers is the total number of containers in the pod
	// 0 if no pod exists
	TotalContainers int
}
