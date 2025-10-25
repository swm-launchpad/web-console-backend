package build_history

import (
	"testing"

	"github.com/stretchr/testify/assert"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

func TestBuildHistoryStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   BuildHistoryStatus
		expected string
	}{
		{"untracked", BuildHistoryStatusUntracked, "untracked"},
		{"backend_trigger_failed", BuildHistoryStatusBackendTriggerFailed, "backend_trigger_failed"},
		{"backend_tracking_failed", BuildHistoryStatusBackendTrackingFailed, "backend_tracking_failed"},
		{"backend_tracking_lost", BuildHistoryStatusBackendTrackingLost, "backend_tracking_lost"},
		{"running", BuildHistoryStatusRunning, "running"},
		{"success", BuildHistoryStatusSuccess, "success"},
		{"failed", BuildHistoryStatusFailed, "failed"},
		{"cancelled", BuildHistoryStatusCancelled, "cancelled"},
		{"skipped", BuildHistoryStatusSkipped, "skipped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestBuildHistoryStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status BuildHistoryStatus
		valid  bool
	}{
		{"untracked", BuildHistoryStatusUntracked, true},
		{"backend_trigger_failed", BuildHistoryStatusBackendTriggerFailed, true},
		{"backend_tracking_failed", BuildHistoryStatusBackendTrackingFailed, true},
		{"backend_tracking_lost", BuildHistoryStatusBackendTrackingLost, true},
		{"running", BuildHistoryStatusRunning, true},
		{"success", BuildHistoryStatusSuccess, true},
		{"failed", BuildHistoryStatusFailed, true},
		{"cancelled", BuildHistoryStatusCancelled, true},
		{"skipped", BuildHistoryStatusSkipped, true},
		{"invalid_status", BuildHistoryStatus("invalid_status"), false},
		{"empty", BuildHistoryStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.status.IsValid())
		})
	}
}

func TestValidateBuildHistoryStatus(t *testing.T) {
	t.Run("valid statuses", func(t *testing.T) {
		validStatuses := []BuildHistoryStatus{
			BuildHistoryStatusUntracked,
			BuildHistoryStatusBackendTriggerFailed,
			BuildHistoryStatusBackendTrackingFailed,
			BuildHistoryStatusBackendTrackingLost,
			BuildHistoryStatusRunning,
			BuildHistoryStatusSuccess,
			BuildHistoryStatusFailed,
			BuildHistoryStatusCancelled,
			BuildHistoryStatusSkipped,
		}

		for _, status := range validStatuses {
			t.Run(status.String(), func(t *testing.T) {
				err := ValidateBuildHistoryStatus(status)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		invalidStatus := BuildHistoryStatus("invalid_status")
		err := ValidateBuildHistoryStatus(invalidStatus)
		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidBuildStatus, err)
	})
}
