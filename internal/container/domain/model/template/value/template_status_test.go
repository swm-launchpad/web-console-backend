package value

import (
	"testing"

	"github.com/stretchr/testify/assert"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

func TestNewTemplateStatus_ValidStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   TemplateStatus
	}{
		{"active", "active", TemplateStatusActive},
		{"inactive", "inactive", TemplateStatusInactive},
		{"deprecated", "deprecated", TemplateStatusDeprecated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTemplateStatus(tt.status)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewTemplateStatus_InvalidStatus(t *testing.T) {
	tests := []string{
		"invalid",
		"Active",
		"ACTIVE",
		"",
		"pending",
	}

	for _, status := range tests {
		t.Run(status, func(t *testing.T) {
			_, err := NewTemplateStatus(status)
			assert.ErrorIs(t, err, containererrors.ErrInvalidTemplateConfig)
		})
	}
}

func TestTemplateStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status TemplateStatus
		want   bool
	}{
		{"active is valid", TemplateStatusActive, true},
		{"inactive is valid", TemplateStatusInactive, true},
		{"deprecated is valid", TemplateStatusDeprecated, true},
		{"invalid status", TemplateStatus("invalid"), false},
		{"empty status", TemplateStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTemplateStatus_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		status TemplateStatus
		want   bool
	}{
		{"active", TemplateStatusActive, true},
		{"inactive", TemplateStatusInactive, false},
		{"deprecated", TemplateStatusDeprecated, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.IsActive()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTemplateStatus_String(t *testing.T) {
	assert.Equal(t, "active", TemplateStatusActive.String())
	assert.Equal(t, "inactive", TemplateStatusInactive.String())
	assert.Equal(t, "deprecated", TemplateStatusDeprecated.String())
}

func TestTemplateStatus_Equals(t *testing.T) {
	assert.True(t, TemplateStatusActive.Equals(TemplateStatusActive))
	assert.False(t, TemplateStatusActive.Equals(TemplateStatusInactive))
	assert.False(t, TemplateStatusInactive.Equals(TemplateStatusDeprecated))
}
