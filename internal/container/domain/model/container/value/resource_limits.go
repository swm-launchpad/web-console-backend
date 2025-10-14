package value

import (
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

// ResourceLimits represents container resource constraints
// It is a value object that encapsulates resource limit validation rules
type ResourceLimits struct {
	cpuLimit    *uint32 // Millicores (1000 = 1 CPU core)
	memoryLimit *uint32 // Mi (Mebibytes)
}

const (
	// CPU limits in millicores
	MinCPULimit = 100  // 0.1 CPU cores
	MaxCPULimit = 4000 // 4 CPU cores

	// Memory limits in Mi
	MinMemoryLimit = 128  // 128 Mi
	MaxMemoryLimit = 8192 // 8 Gi
)

// NewResourceLimits creates a new ResourceLimits with validation
// Both limits are optional (can be nil)
func NewResourceLimits(cpuLimit, memoryLimit *uint32) (ResourceLimits, error) {
	// Validate CPU limit
	if cpuLimit != nil {
		if *cpuLimit < MinCPULimit || *cpuLimit > MaxCPULimit {
			return ResourceLimits{}, containererrors.ErrCPULimitOutOfRange
		}
	}

	// Validate memory limit
	if memoryLimit != nil {
		if *memoryLimit < MinMemoryLimit || *memoryLimit > MaxMemoryLimit {
			return ResourceLimits{}, containererrors.ErrMemoryLimitOutOfRange
		}
	}

	return ResourceLimits{
		cpuLimit:    cpuLimit,
		memoryLimit: memoryLimit,
	}, nil
}

// CPULimit returns the CPU limit in millicores (may be nil)
func (r ResourceLimits) CPULimit() *uint32 {
	return r.cpuLimit
}

// MemoryLimit returns the memory limit in Mi (may be nil)
func (r ResourceLimits) MemoryLimit() *uint32 {
	return r.memoryLimit
}

// CPULimitOrDefault returns the CPU limit or default value
func (r ResourceLimits) CPULimitOrDefault(defaultValue uint32) uint32 {
	if r.cpuLimit != nil {
		return *r.cpuLimit
	}
	return defaultValue
}

// MemoryLimitOrDefault returns the memory limit or default value
func (r ResourceLimits) MemoryLimitOrDefault(defaultValue uint32) uint32 {
	if r.memoryLimit != nil {
		return *r.memoryLimit
	}
	return defaultValue
}

// Equals checks if two ResourceLimits are equal
func (r ResourceLimits) Equals(other ResourceLimits) bool {
	// Compare CPU limits
	if (r.cpuLimit == nil) != (other.cpuLimit == nil) {
		return false
	}
	if r.cpuLimit != nil && *r.cpuLimit != *other.cpuLimit {
		return false
	}

	// Compare memory limits
	if (r.memoryLimit == nil) != (other.memoryLimit == nil) {
		return false
	}
	if r.memoryLimit != nil && *r.memoryLimit != *other.memoryLimit {
		return false
	}

	return true
}

// IsEmpty returns true if both limits are nil
func (r ResourceLimits) IsEmpty() bool {
	return r.cpuLimit == nil && r.memoryLimit == nil
}
