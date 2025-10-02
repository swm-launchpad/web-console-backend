package value

import (
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// ResourceLimits represents the resource limitations for a project as a value object
// All fields are required and must be within valid ranges
type ResourceLimits struct {
	cpuLimit     uint32 // millicores (1000 = 1 CPU)
	memoryLimit  uint32 // Mi (Mebibytes)
	diskLimit    uint32 // Mi (Mebibytes)
	trafficLimit uint32 // Mi (Mebibytes) per month
}

// NewResourceLimits creates a new ResourceLimits with validation
// All parameters are required and must be within valid ranges
func NewResourceLimits(cpu, memoryLimit, disk, traffic uint32) (*ResourceLimits, error) {
	rl := &ResourceLimits{
		cpuLimit:     cpu,
		memoryLimit:  memoryLimit,
		diskLimit:    disk,
		trafficLimit: traffic,
	}

	// Validate the resource limits
	if err := rl.validate(); err != nil {
		return nil, err
	}

	return rl, nil
}

// Unit conversion constants
const (
	MiToBytes = 1024 * 1024 // 1 Mi = 1048576 bytes
	GiBToMi   = 1024        // 1 GiB = 1024 Mi
)

// System-wide absolute resource limits for security and resource management
const (
	// CPU limits in millicores (1000 = 1 CPU core)
	MinCPULimit = 100  // 0.1 cores
	MaxCPULimit = 4000 // 4 cores

	// Memory limits in Mi (Mebibytes)
	MinMemoryLimit = 128  // 128 Mi
	MaxMemoryLimit = 8192 // 8192 Mi = ~8GB

	// Storage limits in Mi (Mebibytes)
	MinDiskLimit = 128          // 128 Mi
	MaxDiskLimit = 10 * GiBToMi // 10 GB = 10240 Mi

	// Traffic limits in Mi (Mebibytes) per month
	MinTrafficLimit = 128                // 128 Mi
	MaxTrafficLimit = 1024 * 1024 * 1024 // 1 TB = 1048576 Mi
)

// validate validates the resource limits
// All fields are required and must be within valid ranges
func (r ResourceLimits) validate() error {
	// CPU Limit validation: min:100 millicores, max:4000 millicores
	if r.cpuLimit < MinCPULimit || r.cpuLimit > MaxCPULimit {
		return projecterrors.ErrCPULimitExceeded
	}

	// Memory Limit validation: min:128Mi, max:8192Mi
	if r.memoryLimit < MinMemoryLimit || r.memoryLimit > MaxMemoryLimit {
		return projecterrors.ErrMemoryLimitExceeded
	}

	// Storage Limit validation: min:128Mi, max:10240Mi
	if r.diskLimit < MinDiskLimit || r.diskLimit > MaxDiskLimit {
		return projecterrors.ErrDiskLimitExceeded
	}

	// Traffic Limit validation: min:128Mi, max:1TB
	if r.trafficLimit < MinTrafficLimit || r.trafficLimit > MaxTrafficLimit {
		return projecterrors.ErrTrafficLimitExceeded
	}

	return nil
}

// CPULimit returns the CPU limit
func (r ResourceLimits) CPULimit() uint32 {
	return r.cpuLimit
}

// MemoryLimit returns the memory limit
func (r ResourceLimits) MemoryLimit() uint32 {
	return r.memoryLimit
}

// DiskLimit returns the disk limit
func (r ResourceLimits) DiskLimit() uint32 {
	return r.diskLimit
}

// TrafficLimit returns the traffic limit
func (r ResourceLimits) TrafficLimit() uint32 {
	return r.trafficLimit
}

// Equals checks if two ResourceLimits are equal
func (r ResourceLimits) Equals(other ResourceLimits) bool {
	return r.cpuLimit == other.cpuLimit &&
		r.memoryLimit == other.memoryLimit &&
		r.diskLimit == other.diskLimit &&
		r.trafficLimit == other.trafficLimit
}
