package value

// DEPRECATED: All pricing constants have been moved to database (SYSTEM_SETTINGS table)
// These constants are kept for backward compatibility during migration
// Use common/settings.SettingsService to retrieve these values instead

// Plan base prices (monthly, in KRW)
// DEPRECATED: Use SettingsService.GetPlanBasePrice()
const (
	FreePlanBasePrice = 0
	EcoPlanBasePrice  = 1100
	ProPlanBasePrice  = 14900
)

// Runtime pricing (per minute, in KRW)
// DEPRECATED: Use SettingsService.GetPlanFreeMinutes() and GetPlanRuntimePricePerMinute()
const (
	// Free plan has unlimited free runtime
	FreePlanFreeMinutes           = -1 // -1 means unlimited
	FreePlanRuntimePricePerMinute = 0.0

	// Eco plan has 500 minutes free per month, then 3.3 KRW per minute
	EcoPlanFreeMinutes           = 500
	EcoPlanRuntimePricePerMinute = 3.3

	// Pro plan has unlimited free runtime (Always On)
	ProPlanFreeMinutes           = -1 // -1 means unlimited
	ProPlanRuntimePricePerMinute = 0.0
)

// Eco plan resource pricing (per minute, in KRW)
// DEPRECATED: Use SettingsService methods (GetEcoCPUPricePerCorePerMinute, etc.)
// These constants are kept for backward compatibility and fallback only
// Updated to match current pricing policy (LP-484)
// Formula: (additional_resource / unit) * pricePerMinute * runtimeMinutes
const (
	EcoCPUPricePerCorePerMinute  = 2.2 // 2.2 KRW per 1 core (1000m) per minute
	EcoMemoryPricePerGBPerMinute = 1.1 // 1.1 KRW per 1GB (1024Mi) per minute
	EcoDiskPricePerGBPerMonth    = 200 // 200 KRW per 1GB (1024Mi) per month (fixed, not per minute)
)

// Pro plan resource pricing (per month, in KRW)
// DEPRECATED: Use SettingsService methods (GetProCPUPricePerCorePerMonth, etc.)
// Formula: (additional_resource / unit) * pricePerMonth
const (
	ProCPUPricePerCorePerMonth  = 5000 // 5000 KRW per 1 core (1000m) per month
	ProMemoryPricePerGBPerMonth = 3000 // 3000 KRW per 1GB (1024Mi) per month
	ProDiskPricePerGBPerMonth   = 200  // 200 KRW per 1GB (1024Mi) per month (same as Eco)
)

// Free tier limits (beta period)
// DEPRECATED: Use SettingsService methods (IsBetaTierEnabled, GetBetaTierCPULimit, etc.)
const (
	FreeTierEnabled     = true
	FreeTierCPULimit    = 2000  // 2 cores (2000m)
	FreeTierMemoryLimit = 4096  // 4GB (4096Mi)
	FreeTierDiskLimit   = 10240 // 10GB (10240Mi)
)

// Plan-specific resource constraints
// DEPRECATED: Use SettingsService methods (GetFreePlanCPULimit, etc.)
const (
	// Free plan has fixed resources (cannot be changed)
	FreePlanCPULimit    = 1000 // 1 core (1000m) - FIXED
	FreePlanMemoryLimit = 2048 // 2GB (2048Mi) - FIXED
	FreePlanDiskLimit   = 1024 // 1GB (1024Mi) - FIXED

	// Free plan project limit per user
	FreePlanMaxProjectsPerUser = 1
)
