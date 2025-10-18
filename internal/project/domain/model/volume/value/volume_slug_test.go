package value

import (
	"testing"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

func TestNewVolumeSlug(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		wantErr error
	}{
		{
			name:    "valid slug with v prefix, timestamp and random",
			slug:    "v2025011812000012345678",
			wantErr: nil,
		},
		{
			name:    "valid slug with all lowercase alphanumeric in random part",
			slug:    "v20250118120000abcdefgh",
			wantErr: nil,
		},
		{
			name:    "valid slug with mixed alphanumeric in random part",
			slug:    "v20250118120000a1b2c3d4",
			wantErr: nil,
		},
		{
			name:    "invalid - too short",
			slug:    "v202501181200001234567",
			wantErr: projecterrors.ErrSlugInvalidLength,
		},
		{
			name:    "invalid - too long",
			slug:    "v20250118120000123456789",
			wantErr: projecterrors.ErrSlugInvalidLength,
		},
		{
			name:    "invalid - missing v prefix",
			slug:    "a2025011812000012345678",
			wantErr: projecterrors.ErrSlugInvalidFormat,
		},
		{
			name:    "invalid - uppercase in random part",
			slug:    "v20250118120000ABCDEFGH",
			wantErr: projecterrors.ErrSlugInvalidFormat,
		},
		{
			name:    "invalid - special characters",
			slug:    "v20250118120000abcd-fgh",
			wantErr: projecterrors.ErrSlugInvalidFormat,
		},
		{
			name:    "invalid - non-numeric timestamp",
			slug:    "v2025011a12000012345678",
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
			slug, err := NewVolumeSlug(tt.slug)
			if err != tt.wantErr {
				t.Errorf("NewVolumeSlug() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && slug.String() != tt.slug {
				t.Errorf("NewVolumeSlug().String() = %v, want %v", slug.String(), tt.slug)
			}
		})
	}
}

func TestVolumeSlug_Equals(t *testing.T) {
	slug1, _ := NewVolumeSlug("v2025011812000012345678")
	slug2, _ := NewVolumeSlug("v2025011812000012345678")
	slug3, _ := NewVolumeSlug("v20250118120000abcdefgh")

	if !slug1.Equals(slug2) {
		t.Error("Expected identical slugs to be equal")
	}

	if slug1.Equals(slug3) {
		t.Error("Expected different slugs to not be equal")
	}
}
