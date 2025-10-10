package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeploymentStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   DeploymentStatus
		expected string
	}{
		{
			name:     "pending status",
			status:   DeploymentStatusPending,
			expected: "pending",
		},
		{
			name:     "running status",
			status:   DeploymentStatusRunning,
			expected: "running",
		},
		{
			name:     "success status",
			status:   DeploymentStatusSuccess,
			expected: "success",
		},
		{
			name:     "failed status",
			status:   DeploymentStatusFailed,
			expected: "failed",
		},
		{
			name:     "cancelled status",
			status:   DeploymentStatusCancelled,
			expected: "cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestDeploymentStatus_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   DeploymentStatus
		expected bool
	}{
		{
			name:     "valid pending status",
			status:   DeploymentStatusPending,
			expected: true,
		},
		{
			name:     "valid running status",
			status:   DeploymentStatusRunning,
			expected: true,
		},
		{
			name:     "valid success status",
			status:   DeploymentStatusSuccess,
			expected: true,
		},
		{
			name:     "valid failed status",
			status:   DeploymentStatusFailed,
			expected: true,
		},
		{
			name:     "valid cancelled status",
			status:   DeploymentStatusCancelled,
			expected: true,
		},
		{
			name:     "invalid status",
			status:   DeploymentStatus("invalid"),
			expected: false,
		},
		{
			name:     "empty status",
			status:   DeploymentStatus(""),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsValid())
		})
	}
}

func TestValidateDeploymentStatus(t *testing.T) {
	tests := []struct {
		name      string
		status    DeploymentStatus
		wantError bool
	}{
		{
			name:      "valid pending status",
			status:    DeploymentStatusPending,
			wantError: false,
		},
		{
			name:      "valid running status",
			status:    DeploymentStatusRunning,
			wantError: false,
		},
		{
			name:      "valid success status",
			status:    DeploymentStatusSuccess,
			wantError: false,
		},
		{
			name:      "valid failed status",
			status:    DeploymentStatusFailed,
			wantError: false,
		},
		{
			name:      "valid cancelled status",
			status:    DeploymentStatusCancelled,
			wantError: false,
		},
		{
			name:      "invalid status",
			status:    DeploymentStatus("invalid"),
			wantError: true,
		},
		{
			name:      "empty status",
			status:    DeploymentStatus(""),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDeploymentStatus(tt.status)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
