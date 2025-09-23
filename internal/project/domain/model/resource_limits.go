package model

import (
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// ResourceLimits represents the resource limitations for a project as a value object
type ResourceLimits struct {
	cpuLimit     *uint32 // millicores (1000 = 1 CPU)
	memoryLimit  *uint32 // MB
	diskLimit    *uint32 // GB
	trafficLimit *uint64 // GB per month
}

// ResourceUsage represents current resource usage
type ResourceUsage struct {
	CPUUsage     uint32 // millicores
	MemoryUsage  uint32 // MB
	DiskUsage    uint32 // GB
	TrafficUsage uint64 // GB
}

// PlanLimits defines resource limits for different plans
var PlanLimits = map[string]ResourceLimits{
	"free": {
		cpuLimit:     uint32Ptr(500), // 0.5 CPU
		memoryLimit:  uint32Ptr(512), // 512 MB
		diskLimit:    uint32Ptr(1),   // 1 GB
		trafficLimit: uint64Ptr(10),  // 10 GB/month
	},
	"starter": {
		cpuLimit:     uint32Ptr(1000), // 1 CPU
		memoryLimit:  uint32Ptr(2048), // 2 GB
		diskLimit:    uint32Ptr(10),   // 10 GB
		trafficLimit: uint64Ptr(100),  // 100 GB/month
	},
	"pro": {
		cpuLimit:     uint32Ptr(2000), // 2 CPU
		memoryLimit:  uint32Ptr(4096), // 4 GB
		diskLimit:    uint32Ptr(50),   // 50 GB
		trafficLimit: uint64Ptr(500),  // 500 GB/month
	},
	"enterprise": {
		cpuLimit:     uint32Ptr(8000),  // 8 CPU
		memoryLimit:  uint32Ptr(16384), // 16 GB
		diskLimit:    uint32Ptr(500),   // 500 GB
		trafficLimit: uint64Ptr(5000),  // 5000 GB/month
	},
}

// NewResourceLimits creates a new ResourceLimits with validation
func NewResourceLimits(cpu, memory, disk *uint32, traffic *uint64) (*ResourceLimits, error) {
	// Validate that values are non-negative (nil is allowed for unlimited)
	if cpu != nil && *cpu == 0 {
		return nil, projecterrors.ErrCPULimitNegative
	}
	if memory != nil && *memory == 0 {
		return nil, projecterrors.ErrMemoryLimitNegative
	}
	if disk != nil && *disk == 0 {
		return nil, projecterrors.ErrDiskLimitNegative
	}
	if traffic != nil && *traffic == 0 {
		return nil, projecterrors.ErrTrafficLimitNegative
	}

	return &ResourceLimits{
		cpuLimit:     cpu,
		memoryLimit:  memory,
		diskLimit:    disk,
		trafficLimit: traffic,
	}, nil
}

// NewResourceLimitsForPlan creates ResourceLimits based on a plan name
func NewResourceLimitsForPlan(plan string) (*ResourceLimits, error) {
	limits, exists := PlanLimits[plan]
	if !exists {
		return nil, projecterrors.ErrPlanNotFound
	}

	// Create a copy to avoid modifying the global plan limits
	return &ResourceLimits{
		cpuLimit:     copyUint32Ptr(limits.cpuLimit),
		memoryLimit:  copyUint32Ptr(limits.memoryLimit),
		diskLimit:    copyUint32Ptr(limits.diskLimit),
		trafficLimit: copyUint64Ptr(limits.trafficLimit),
	}, nil
}

// Validate validates the resource limits
func (r ResourceLimits) Validate() error {
	// All limits are optional (nil means unlimited)
	// But if set, they must be positive
	if r.cpuLimit != nil && *r.cpuLimit == 0 {
		return projecterrors.ErrCPULimitNegative
	}
	if r.memoryLimit != nil && *r.memoryLimit == 0 {
		return projecterrors.ErrMemoryLimitNegative
	}
	if r.diskLimit != nil && *r.diskLimit == 0 {
		return projecterrors.ErrDiskLimitNegative
	}
	if r.trafficLimit != nil && *r.trafficLimit == 0 {
		return projecterrors.ErrTrafficLimitNegative
	}
	return nil
}

// ValidateForPlan validates if the resource limits are within the plan's constraints
func (r ResourceLimits) ValidateForPlan(plan string) error {
	planLimits, exists := PlanLimits[plan]
	if !exists {
		return projecterrors.ErrPlanNotFound
	}

	if r.Exceeds(planLimits) {
		return projecterrors.ErrPlanLimitExceeded
	}

	return nil
}

// Exceeds checks if current limits exceed the other limits
func (r ResourceLimits) Exceeds(other ResourceLimits) bool {
	// If other has a limit and we exceed it, return true
	if other.cpuLimit != nil && r.cpuLimit != nil && *r.cpuLimit > *other.cpuLimit {
		return true
	}
	if other.memoryLimit != nil && r.memoryLimit != nil && *r.memoryLimit > *other.memoryLimit {
		return true
	}
	if other.diskLimit != nil && r.diskLimit != nil && *r.diskLimit > *other.diskLimit {
		return true
	}
	if other.trafficLimit != nil && r.trafficLimit != nil && *r.trafficLimit > *other.trafficLimit {
		return true
	}

	return false
}

// IsWithinQuota checks if the usage is within the resource limits
func (r ResourceLimits) IsWithinQuota(usage ResourceUsage) bool {
	if r.cpuLimit != nil && usage.CPUUsage > *r.cpuLimit {
		return false
	}
	if r.memoryLimit != nil && usage.MemoryUsage > *r.memoryLimit {
		return false
	}
	if r.diskLimit != nil && usage.DiskUsage > *r.diskLimit {
		return false
	}
	if r.trafficLimit != nil && usage.TrafficUsage > *r.trafficLimit {
		return false
	}

	return true
}

// GetCPULimit returns the CPU limit
func (r ResourceLimits) GetCPULimit() *uint32 {
	return copyUint32Ptr(r.cpuLimit)
}

// GetMemoryLimit returns the memory limit
func (r ResourceLimits) GetMemoryLimit() *uint32 {
	return copyUint32Ptr(r.memoryLimit)
}

// GetDiskLimit returns the disk limit
func (r ResourceLimits) GetDiskLimit() *uint32 {
	return copyUint32Ptr(r.diskLimit)
}

// GetTrafficLimit returns the traffic limit
func (r ResourceLimits) GetTrafficLimit() *uint64 {
	return copyUint64Ptr(r.trafficLimit)
}

// Equals checks if two ResourceLimits are equal
func (r ResourceLimits) Equals(other ResourceLimits) bool {
	if !equalUint32Ptr(r.cpuLimit, other.cpuLimit) {
		return false
	}
	if !equalUint32Ptr(r.memoryLimit, other.memoryLimit) {
		return false
	}
	if !equalUint32Ptr(r.diskLimit, other.diskLimit) {
		return false
	}
	if !equalUint64Ptr(r.trafficLimit, other.trafficLimit) {
		return false
	}

	return true
}

// IsUnlimited checks if all resource limits are unlimited (nil)
func (r ResourceLimits) IsUnlimited() bool {
	return r.cpuLimit == nil && r.memoryLimit == nil &&
		r.diskLimit == nil && r.trafficLimit == nil
}

// HasCPULimit checks if CPU limit is set
func (r ResourceLimits) HasCPULimit() bool {
	return r.cpuLimit != nil
}

// HasMemoryLimit checks if memory limit is set
func (r ResourceLimits) HasMemoryLimit() bool {
	return r.memoryLimit != nil
}

// HasDiskLimit checks if disk limit is set
func (r ResourceLimits) HasDiskLimit() bool {
	return r.diskLimit != nil
}

// HasTrafficLimit checks if traffic limit is set
func (r ResourceLimits) HasTrafficLimit() bool {
	return r.trafficLimit != nil
}

// Helper functions

func uint32Ptr(v uint32) *uint32 {
	return &v
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}

func copyUint32Ptr(p *uint32) *uint32 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func copyUint64Ptr(p *uint64) *uint64 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func equalUint32Ptr(a, b *uint32) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func equalUint64Ptr(a, b *uint64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
