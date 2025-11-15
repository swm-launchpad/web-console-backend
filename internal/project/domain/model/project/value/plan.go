package value

import (
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// Plan represents the project plan type
type Plan string

const (
	PlanFree Plan = "free"
	PlanEco  Plan = "eco"
	PlanPro  Plan = "pro"
)

// NewPlan creates a new Plan with validation
func NewPlan(planStr string) (Plan, error) {
	plan := Plan(planStr)
	if !plan.IsValid() {
		return "", projecterrors.ErrInvalidPlan
	}
	return plan, nil
}

// IsValid checks if the plan is valid
func (p Plan) IsValid() bool {
	switch p {
	case PlanFree, PlanEco, PlanPro:
		return true
	default:
		return false
	}
}

// IsScaleToZero returns true if the plan supports scale to zero
// Free and Eco plans support scale to zero
func (p Plan) IsScaleToZero() bool {
	return p == PlanFree || p == PlanEco
}

// IsAlwaysOn returns true if the plan is always on (no scale to zero)
// Pro plan is always on
func (p Plan) IsAlwaysOn() bool {
	return p == PlanPro
}

// HasAdvertisement returns true if the plan shows advertisements
// Only Free plan shows advertisements during scale to zero
func (p Plan) HasAdvertisement() bool {
	return p == PlanFree
}

// IsUsageBased returns true if the plan uses usage-based pricing
// Free and Eco plans use usage-based pricing (charged by runtime hours)
func (p Plan) IsUsageBased() bool {
	return p == PlanFree || p == PlanEco
}

// IsFixedPrice returns true if the plan uses fixed pricing
// Pro plan uses fixed monthly pricing
func (p Plan) IsFixedPrice() bool {
	return p == PlanPro
}

// String returns the string representation of the plan
func (p Plan) String() string {
	return string(p)
}

// CanAddResources returns true if the plan allows adding resources beyond the base allocation
// Free plan has fixed resources, Eco and Pro can add resources
func (p Plan) CanAddResources() bool {
	return p == PlanEco || p == PlanPro
}

// HasFixedResources returns true if the plan has fixed resources (cannot be changed)
// Only Free plan has fixed resources
func (p Plan) HasFixedResources() bool {
	return p == PlanFree
}

// GetDefaultCPULimit returns the default CPU limit for the plan
// DEPRECATED: Not used. Handler uses hardcoded constants, ProjectService validates against DB settings.
func (p Plan) GetDefaultCPULimit() uint32 {
	// All plans now have 1 core (1000m) as default
	return DefaultCPULimit // 1000
}

// GetDefaultMemoryLimit returns the default memory limit for the plan
// DEPRECATED: Not used. Handler uses hardcoded constants, ProjectService validates against DB settings.
func (p Plan) GetDefaultMemoryLimit() uint32 {
	// All plans now have 2GB (2048Mi) as default
	return DefaultMemoryLimit // 2048
}

// GetDefaultDiskLimit returns the default disk limit for the plan
// DEPRECATED: Not used. Handler uses hardcoded constants, ProjectService validates against DB settings.
func (p Plan) GetDefaultDiskLimit() uint32 {
	switch p {
	case PlanFree:
		return 1024 // 1GB
	case PlanEco:
		return 2048 // 2GB
	case PlanPro:
		return 10240 // 10GB
	default:
		return DefaultDiskLimit
	}
}

// GetBasePrice returns the monthly base price for the plan (in KRW)
// DEPRECATED: Use SettingsService.GetPlanBasePrice() instead. This uses hardcoded constants.
func (p Plan) GetBasePrice() int {
	switch p {
	case PlanFree:
		return FreePlanBasePrice
	case PlanEco:
		return EcoPlanBasePrice
	case PlanPro:
		return ProPlanBasePrice
	default:
		return 0
	}
}

// GetFreeMinutes returns the free runtime minutes per month for the plan
// Returns -1 for unlimited
// DEPRECATED: Use SettingsService.GetPlanFreeMinutes() instead. This uses hardcoded constants.
func (p Plan) GetFreeMinutes() int {
	switch p {
	case PlanFree:
		return FreePlanFreeMinutes
	case PlanEco:
		return EcoPlanFreeMinutes
	case PlanPro:
		return ProPlanFreeMinutes
	default:
		return 0
	}
}

// GetRuntimePricePerMinute returns the price per minute after free minutes are exhausted (in KRW)
func (p Plan) GetRuntimePricePerMinute() float64 {
	switch p {
	case PlanFree:
		return FreePlanRuntimePricePerMinute
	case PlanEco:
		return EcoPlanRuntimePricePerMinute
	case PlanPro:
		return ProPlanRuntimePricePerMinute
	default:
		return 0
	}
}
