package value

import (
	"testing"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

func TestNewProjectSlug(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		wantErr error
	}{
		{
			name:    "valid slug with p prefix, timestamp and random",
			slug:    "p2025011812000012345678",
			wantErr: nil,
		},
		{
			name:    "valid slug with all lowercase alphanumeric in random part",
			slug:    "p20250118120000abcdefgh",
			wantErr: nil,
		},
		{
			name:    "valid slug with mixed alphanumeric in random part",
			slug:    "p20250118120000a1b2c3d4",
			wantErr: nil,
		},
		{
			name:    "invalid - too short",
			slug:    "p202501181200001234567",
			wantErr: projecterrors.ErrSlugInvalidLength,
		},
		{
			name:    "invalid - too long",
			slug:    "p20250118120000123456789",
			wantErr: projecterrors.ErrSlugInvalidLength,
		},
		{
			name:    "invalid - missing p prefix",
			slug:    "a2025011812000012345678",
			wantErr: projecterrors.ErrSlugInvalidFormat,
		},
		{
			name:    "invalid - uppercase in random part",
			slug:    "p20250118120000ABCDEFGH",
			wantErr: projecterrors.ErrSlugInvalidFormat,
		},
		{
			name:    "invalid - special characters",
			slug:    "p20250118120000abcd-fgh",
			wantErr: projecterrors.ErrSlugInvalidFormat,
		},
		{
			name:    "invalid - non-numeric timestamp",
			slug:    "p2025011a12000012345678",
			wantErr: projecterrors.ErrSlugInvalidFormat,
		},
		{
			name:    "invalid - empty string",
			slug:    "",
			wantErr: projecterrors.ErrSlugInvalidLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug, err := NewProjectSlug(tt.slug)
			if err != tt.wantErr {
				t.Errorf("NewProjectSlug() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && slug.String() != tt.slug {
				t.Errorf("NewProjectSlug().String() = %v, want %v", slug.String(), tt.slug)
			}
		})
	}
}

func TestProjectSlug_Equals(t *testing.T) {
	slug1, _ := NewProjectSlug("p2025011812000012345678")
	slug2, _ := NewProjectSlug("p2025011812000012345678")
	slug3, _ := NewProjectSlug("p20250118120000abcdefgh")

	if !slug1.Equals(*slug2) {
		t.Error("Expected identical slugs to be equal")
	}

	if slug1.Equals(*slug3) {
		t.Error("Expected different slugs to not be equal")
	}
}
