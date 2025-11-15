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

// Default resource allocation for all plans (base included resources)
// Note: Pro plan has higher disk allocation (10GB) defined in handler
const (
	DefaultCPULimit    = 1000 // 1 core (1000 millicores)
	DefaultMemoryLimit = 2048 // 2GB (2048 Mi)
	DefaultDiskLimit   = 1024 // 1GB (1024 Mi) - Free plan, Eco uses 2GB, Pro uses 10GB
)

// System-wide absolute resource limits for security and resource management
const (
	// CPU limits in millicores (1000 = 1 CPU core)
	// Step: 500 millicores (0.5 core)
	MinCPULimit = 500  // 0.5 cores
	MaxCPULimit = 8000 // 8 cores

	// Memory limits in Mi (Mebibytes)
	// Step: 512 Mi (0.5 GB)
	MinMemoryLimit = 512   // 512 Mi = 0.5GB
	MaxMemoryLimit = 16384 // 16384 Mi = 16GB

	// Storage limits in Mi (Mebibytes)
	// Step: 512 Mi (0.5 GB)
	MinDiskLimit = 1024         // 1024 Mi = 1GB
	MaxDiskLimit = 32 * GiBToMi // 32 GB = 32768 Mi

	// Traffic limits in Mi (Mebibytes) per month
	// Traffic is not pre-allocated, maintained for backward compatibility
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
