package value

import projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"

// ProjectOperationStatus represents the current operation status of a project
type ProjectOperationStatus string

const (
	// ProjectOperationStatusNothing indicates no operation is currently running
	ProjectOperationStatusNothing ProjectOperationStatus = "nothing"

	// ProjectOperationStatusBuilding indicates a build operation is in progress
	ProjectOperationStatusBuilding ProjectOperationStatus = "building"

	// ProjectOperationStatusDeploying indicates a deployment operation is in progress
	ProjectOperationStatusDeploying ProjectOperationStatus = "deploying"
)

// String returns the string representation of the operation status
func (s ProjectOperationStatus) String() string {
	return string(s)
}

// IsValid checks if the operation status is valid
func (s ProjectOperationStatus) IsValid() bool {
	switch s {
	case ProjectOperationStatusNothing,
		ProjectOperationStatusBuilding,
		ProjectOperationStatusDeploying:
		return true
	default:
		return false
	}
}

// ValidateProjectOperationStatus validates the operation status
func ValidateProjectOperationStatus(s ProjectOperationStatus) error {
	if !s.IsValid() {
		return projecterrors.ErrInvalidStatusTransition
	}
	return nil
}
