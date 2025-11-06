// Package dto defines Data Transfer Objects for infrastructure layer communications.
package dto

// PodStatus represents the status of a Kubernetes Pod for a project.
// This is used to check if application pods are running and healthy.
type PodStatus struct {
	// Exists indicates whether at least one pod exists for the project
	Exists bool

	// Phase is the current phase of the pod (e.g., "Running", "Pending", "Failed")
	// Empty if no pod exists
	Phase string

	// ReadyContainers is the number of containers that are ready
	// 0 if no pod exists
	ReadyContainers int

	// TotalContainers is the total number of containers in the pod
	// 0 if no pod exists
	TotalContainers int
}
