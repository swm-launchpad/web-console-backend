package model

import (
	"testing"
	"time"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

func TestNewMount(t *testing.T) {
	tests := []struct {
		name        string
		containerID uint
		volumeID    uint
		mountPath   string
		wantErr     error
	}{
		{
			name:        "Valid mount",
			containerID: 1,
			volumeID:    10,
			mountPath:   "/app/data",
			wantErr:     nil,
		},
		{
			name:        "Invalid container ID",
			containerID: 0,
			volumeID:    10,
			mountPath:   "/app/data",
			wantErr:     containererrors.ErrInvalidContainerID,
		},
		{
			name:        "Invalid volume ID",
			containerID: 1,
			volumeID:    0,
			mountPath:   "/app/data",
			wantErr:     containererrors.ErrInvalidVolumeID,
		},
		{
			name:        "Empty mount path",
			containerID: 1,
			volumeID:    10,
			mountPath:   "",
			wantErr:     containererrors.ErrMountPathRequired,
		},
		{
			name:        "Mount path too long",
			containerID: 1,
			volumeID:    10,
			mountPath:   "/" + string(make([]byte, MaxMountPathLength)),
			wantErr:     containererrors.ErrMountPathTooLong,
		},
		{
			name:        "Non-absolute mount path",
			containerID: 1,
			volumeID:    10,
			mountPath:   "app/data",
			wantErr:     containererrors.ErrInvalidMountFormat,
		},
		{
			name:        "System path /bin",
			containerID: 1,
			volumeID:    10,
			mountPath:   "/bin",
			wantErr:     containererrors.ErrMountPathReserved,
		},
		{
			name:        "System path /etc/config",
			containerID: 1,
			volumeID:    10,
			mountPath:   "/etc/config",
			wantErr:     containererrors.ErrMountPathReserved,
		},
		{
			name:        "Valid home directory path",
			containerID: 1,
			volumeID:    10,
			mountPath:   "/home/user/data",
			wantErr:     nil,
		},
		{
			name:        "Valid tmp path",
			containerID: 1,
			volumeID:    10,
			mountPath:   "/tmp/uploads",
			wantErr:     nil,
		},
		{
			name:        "Valid var path",
			containerID: 1,
			volumeID:    10,
			mountPath:   "/var/www/html",
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mount, err := NewMount(tt.containerID, tt.volumeID, tt.mountPath)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("NewMount() expected error %v, got nil", tt.wantErr)
					return
				}
				if err != tt.wantErr {
					t.Errorf("NewMount() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("NewMount() unexpected error = %v", err)
				return
			}

			if mount == nil {
				t.Error("NewMount() returned nil mount")
				return
			}

			if mount.ContainerID() != tt.containerID {
				t.Errorf("mount.ContainerID() = %v, want %v", mount.ContainerID(), tt.containerID)
			}
			if mount.VolumeID() != tt.volumeID {
				t.Errorf("mount.VolumeID() = %v, want %v", mount.VolumeID(), tt.volumeID)
			}
			if mount.MountPath() != tt.mountPath {
				t.Errorf("mount.MountPath() = %v, want %v", mount.MountPath(), tt.mountPath)
			}
			if mount.CreatedAt().IsZero() {
				t.Error("mount.CreatedAt() is zero")
			}
		})
	}
}

func TestMount_UpdateMountPath(t *testing.T) {
	mount, _ := NewMount(1, 10, "/app/data")

	tests := []struct {
		name    string
		newPath string
		wantErr error
	}{
		{
			name:    "Valid update",
			newPath: "/app/uploads",
			wantErr: nil,
		},
		{
			name:    "Empty path",
			newPath: "",
			wantErr: containererrors.ErrMountPathRequired,
		},
		{
			name:    "Too long path",
			newPath: "/" + string(make([]byte, MaxMountPathLength)),
			wantErr: containererrors.ErrMountPathTooLong,
		},
		{
			name:    "Non-absolute path",
			newPath: "relative/path",
			wantErr: containererrors.ErrInvalidMountFormat,
		},
		{
			name:    "System path",
			newPath: "/bin/bash",
			wantErr: containererrors.ErrMountPathReserved,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mount.UpdateMountPath(tt.newPath)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("UpdateMountPath() expected error %v, got nil", tt.wantErr)
					return
				}
				if err != tt.wantErr {
					t.Errorf("UpdateMountPath() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("UpdateMountPath() unexpected error = %v", err)
				return
			}

			if mount.MountPath() != tt.newPath {
				t.Errorf("mount.MountPath() = %v, want %v", mount.MountPath(), tt.newPath)
			}
		})
	}
}

func TestMount_Equals(t *testing.T) {
	mount1, _ := NewMount(1, 10, "/app/data")
	mount2, _ := NewMount(1, 10, "/different/path")
	mount3, _ := NewMount(1, 20, "/app/data")

	if !mount1.Equals(mount2) {
		t.Error("Mounts with same volume ID should be equal")
	}

	if mount1.Equals(mount3) {
		t.Error("Mounts with different volume IDs should not be equal")
	}

	if mount1.Equals(nil) {
		t.Error("Mount should not equal nil")
	}
}

func TestReconstructMount(t *testing.T) {
	containerID := uint(1)
	volumeID := uint(10)
	mountPath := "/app/data"
	createdAt := time.Now()
	updatedAt := time.Now()

	mount := ReconstructMount(containerID, volumeID, mountPath, createdAt, updatedAt)

	if mount == nil {
		t.Fatal("ReconstructMount() returned nil")
	}

	if mount.ContainerID() != containerID {
		t.Errorf("mount.ContainerID() = %v, want %v", mount.ContainerID(), containerID)
	}
	if mount.VolumeID() != volumeID {
		t.Errorf("mount.VolumeID() = %v, want %v", mount.VolumeID(), volumeID)
	}
	if mount.MountPath() != mountPath {
		t.Errorf("mount.MountPath() = %v, want %v", mount.MountPath(), mountPath)
	}
	if !mount.CreatedAt().Equal(createdAt) {
		t.Errorf("mount.CreatedAt() = %v, want %v", mount.CreatedAt(), createdAt)
	}
	if !mount.UpdatedAt().Equal(updatedAt) {
		t.Errorf("mount.UpdatedAt() = %v, want %v", mount.UpdatedAt(), updatedAt)
	}
}
