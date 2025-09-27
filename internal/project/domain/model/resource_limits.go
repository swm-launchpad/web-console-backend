package model

import (
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// ResourceLimits represents the resource limitations for a project as a value object
type ResourceLimits struct {
	cpuLimit      *uint32 // millicores (1000 = 1 CPU)
	memoryRequest *uint32 // Mi (Mebibytes)
	memoryLimit   *uint32 // Mi (Mebibytes)
	diskLimit     *uint32 // Mi (Mebibytes)
	trafficLimit  *uint64 // Mi (Mebibytes) per month, nil = unlimited
}

// ResourceUsage represents current resource usage
type ResourceUsage struct {
	CPUUsage         uint32 // millicores
	MemoryReqUsage   uint32 // Mi (Mebibytes)
	MemoryLimitUsage uint32 // Mi (Mebibytes)
	DiskUsage        uint32 // Mi (Mebibytes)
	TrafficUsage     uint64 // Mi (Mebibytes)
}

// PlanLimits - Plan-based limits are not implemented yet (per user request)
// Plans should be set to null in database until plan feature is implemented
// var PlanLimits = map[string]ResourceLimits{...}

// NewResourceLimits creates a new ResourceLimits with validation
func NewResourceLimits(cpu *uint32, memoryRequest, memoryLimit, disk *uint32, traffic *uint64) (*ResourceLimits, error) {
	rl := &ResourceLimits{
		cpuLimit:      cpu,
		memoryRequest: memoryRequest,
		memoryLimit:   memoryLimit,
		diskLimit:     disk,
		trafficLimit:  traffic,
	}

	// Validate the resource limits
	if err := rl.Validate(); err != nil {
		return nil, err
	}

	return rl, nil
}

// NewResourceLimitsForPlan - Plan-based limits are not implemented yet (per user request)
// func NewResourceLimitsForPlan(plan string) (*ResourceLimits, error) { ... }

// Unit conversion constants
const (
	MiToBytes = 1024 * 1024 // 1 Mi = 1048576 bytes
	GiBToMi   = 1024        // 1 GiB = 1024 Mi
)

// System-wide absolute resource limits for security and resource management
const (
	// CPU limits in millicores (1000 = 1 CPU core)
	MinCPULimit = 0
	MaxCPULimit = 4000 // 4 cores

	// Memory limits in Mi (Mebibytes)
	MinMemoryRequest = 128  // 128 Mi
	MinMemoryLimit   = 128  // 128 Mi
	MaxMemoryLimit   = 8192 // 8192 Mi = ~8GB

	// Storage limits in Mi (Mebibytes)
	MinDiskLimit = 128          // 128 Mi
	MaxDiskLimit = 10 * GiBToMi // 10 GB = 10240 Mi

	// Traffic limits in Mi (Mebibytes) per month
	MinTrafficLimit = 128 // 128 Mi
	// MaxTrafficLimit = nil (unlimited)
)

// Validate validates the resource limits
func (r ResourceLimits) Validate() error {
	// CPU Limit validation: min:0, max:4000 millicores
	if r.cpuLimit != nil {
		if *r.cpuLimit < MinCPULimit || *r.cpuLimit > MaxCPULimit {
			return projecterrors.ErrCPULimitExceeded
		}
	}

	// Memory Request validation: min:128Mi, max: Memory Limit (if set)
	if r.memoryRequest != nil {
		if *r.memoryRequest < MinMemoryRequest {
			return projecterrors.ErrMemoryRequestTooSmall
		}
		// If memory limit is also set, request cannot exceed limit
		if r.memoryLimit != nil && *r.memoryRequest > *r.memoryLimit {
			return projecterrors.ErrMemoryRequestExceedsLimit
		}
	}

	// Memory Limit validation: min:128Mi, max:8192Mi
	if r.memoryLimit != nil {
		if *r.memoryLimit < MinMemoryLimit || *r.memoryLimit > MaxMemoryLimit {
			return projecterrors.ErrMemoryLimitExceeded
		}
	}

	// Storage Limit validation: min:128Mi, max:10240Mi
	if r.diskLimit != nil {
		if *r.diskLimit < MinDiskLimit || *r.diskLimit > MaxDiskLimit {
			return projecterrors.ErrDiskLimitExceeded
		}
	}

	// Traffic Limit validation: min:128Mi, max:unlimited
	if r.trafficLimit != nil {
		if *r.trafficLimit < MinTrafficLimit {
			return projecterrors.ErrTrafficLimitTooSmall
		}
		// No maximum limit for traffic (unlimited)
	}

	return nil
}

// ValidateForPlan - Plan validation not implemented yet (per user request)
// func (r ResourceLimits) ValidateForPlan(plan string) error { ... }

// Exceeds checks if current limits exceed the other limits
func (r ResourceLimits) Exceeds(other ResourceLimits) bool {
	// If other has a limit and we exceed it, return true
	if other.cpuLimit != nil && r.cpuLimit != nil && *r.cpuLimit > *other.cpuLimit {
		return true
	}
	if other.memoryRequest != nil && r.memoryRequest != nil && *r.memoryRequest > *other.memoryRequest {
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
	if r.memoryRequest != nil && usage.MemoryReqUsage > *r.memoryRequest {
		return false
	}
	if r.memoryLimit != nil && usage.MemoryLimitUsage > *r.memoryLimit {
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

// GetMemoryRequest returns the memory request
func (r ResourceLimits) GetMemoryRequest() *uint32 {
	return copyUint32Ptr(r.memoryRequest)
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
	if !equalUint32Ptr(r.memoryRequest, other.memoryRequest) {
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
	return r.cpuLimit == nil && r.memoryRequest == nil && r.memoryLimit == nil &&
		r.diskLimit == nil && r.trafficLimit == nil
}

// HasCPULimit checks if CPU limit is set
func (r ResourceLimits) HasCPULimit() bool {
	return r.cpuLimit != nil
}

// HasMemoryRequest checks if memory request is set
func (r ResourceLimits) HasMemoryRequest() bool {
	return r.memoryRequest != nil
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
