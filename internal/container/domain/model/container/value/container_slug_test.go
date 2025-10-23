package value

import (
	"testing"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

func TestNewContainerSlug(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		wantErr error
	}{
		{
			name:    "valid slug with c prefix, timestamp and random",
			slug:    "c2025011812000012345678",
			wantErr: nil,
		},
		{
			name:    "valid slug with all lowercase alphanumeric in random part",
			slug:    "c20250118120000abcdefgh",
			wantErr: nil,
		},
		{
			name:    "valid slug with mixed alphanumeric in random part",
			slug:    "c20250118120000a1b2c3d4",
			wantErr: nil,
		},
		{
			name:    "invalid - too short",
			slug:    "c202501181200001234567",
			wantErr: containererrors.ErrSlugInvalidLength,
		},
		{
			name:    "invalid - too long",
			slug:    "c20250118120000123456789",
			wantErr: containererrors.ErrSlugInvalidLength,
		},
		{
			name:    "invalid - missing c prefix",
			slug:    "a2025011812000012345678",
			wantErr: containererrors.ErrSlugInvalidFormat,
		},
		{
			name:    "invalid - uppercase in random part",
			slug:    "c20250118120000ABCDEFGH",
			wantErr: containererrors.ErrSlugInvalidFormat,
		},
		{
			name:    "invalid - special characters",
			slug:    "c20250118120000abcd-fgh",
			wantErr: containererrors.ErrSlugInvalidFormat,
		},
		{
			name:    "invalid - non-numeric timestamp",
			slug:    "c2025011a12000012345678",
			wantErr: containererrors.ErrSlugInvalidFormat,
		},
		{
			name:    "invalid - empty string",
			slug:    "",
			wantErr: containererrors.ErrSlugInvalidLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug, err := NewContainerSlug(tt.slug)
			if err != tt.wantErr {
				t.Errorf("NewContainerSlug() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && slug.String() != tt.slug {
				t.Errorf("NewContainerSlug().String() = %v, want %v", slug.String(), tt.slug)
			}
		})
	}
}

func TestContainerSlug_Equals(t *testing.T) {
	slug1, _ := NewContainerSlug("c2025011812000012345678")
	slug2, _ := NewContainerSlug("c2025011812000012345678")
	slug3, _ := NewContainerSlug("c20250118120000abcdefgh")

	if !slug1.Equals(slug2) {
		t.Error("Expected identical slugs to be equal")
	}

	if slug1.Equals(slug3) {
		t.Error("Expected different slugs to not be equal")
	}
}
