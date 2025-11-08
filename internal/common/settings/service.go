package settings

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// SettingsService defines the interface for settings business logic
type SettingsService interface {
	// Pricing methods
	GetPlanBasePrice(plan string) (int, error)
	GetPlanFreeMinutes(plan string) (int, error)
	GetPlanRuntimePricePerMinute(plan string) (float64, error)

	// Resource pricing - Eco plan (per minute)
	GetEcoCPUPricePerCorePerMinute() (float64, error)
	GetEcoMemoryPricePerGBPerMinute() (float64, error)
	GetEcoDiskPricePerGBPerMonth() (int, error)

	// Resource pricing - Pro plan (per month)
	GetProCPUPricePerCorePerMonth() (int, error)
	GetProMemoryPricePerGBPerMonth() (int, error)
	GetProDiskPricePerGBPerMonth() (int, error)

	// Free plan limits
	GetFreePlanCPULimit() (int, error)
	GetFreePlanMemoryLimit() (int, error)
	GetFreePlanDiskLimit() (int, error)
	GetFreePlanMaxProjects() (int, error)

	// Project count limits
	GetMaxProjectsPerUser() (int, error)

	// Beta tier limits
	IsBetaTierEnabled() (bool, error)
	GetBetaTierCPULimit() (int, error)
	GetBetaTierMemoryLimit() (int, error)
	GetBetaTierDiskLimit() (int, error)

	// Cache management
	InvalidateCache(key string)
	InvalidateAll()

	// Direct access methods for handlers
	GetByKey(key string) (*Setting, error)
	GetByCategory(category string) ([]*Setting, error)
	GetAll() ([]*Setting, error)
	UpdateSetting(key, value string, updatedBy *int) error
}

// cachedValue holds a cached setting value with expiration time
type cachedValue struct {
	value     string
	expiresAt time.Time
}

// settingsService is the concrete implementation with in-memory caching
type settingsService struct {
	repo  SettingsRepository
	cache sync.Map // concurrent-safe map
	ttl   time.Duration
}

// NewSettingsService creates a new instance of SettingsService with 1-minute cache TTL
func NewSettingsService(repo SettingsRepository) SettingsService {
	return &settingsService{
		repo: repo,
		ttl:  1 * time.Minute, // 1-minute cache TTL
	}
}

// getFromCacheOrDB retrieves a setting from cache or database
// Returns error if setting not found (fail-fast approach, no fallback)
func (s *settingsService) getFromCacheOrDB(key string) (string, error) {
	// 1. Check cache
	if val, ok := s.cache.Load(key); ok {
		cached := val.(cachedValue)
		if time.Now().Before(cached.expiresAt) {
			return cached.value, nil
		}
		// Cache expired, remove it
		s.cache.Delete(key)
	}

	// 2. Fetch from database
	setting, err := s.repo.GetByKey(context.Background(), key)
	if err != nil {
		return "", fmt.Errorf("required setting '%s' not found in database: %w", key, err)
	}

	// 3. Store in cache
	s.cache.Store(key, cachedValue{
		value:     setting.Value,
		expiresAt: time.Now().Add(s.ttl),
	})

	return setting.Value, nil
}

// InvalidateCache invalidates a specific key in the cache
func (s *settingsService) InvalidateCache(key string) {
	s.cache.Delete(key)
}

// InvalidateAll clears the entire cache
func (s *settingsService) InvalidateAll() {
	s.cache.Range(func(key, value interface{}) bool {
		s.cache.Delete(key)
		return true
	})
}

// GetInt retrieves an integer setting value
func (s *settingsService) getInt(key string) (int, error) {
	val, err := s.getFromCacheOrDB(key)
	if err != nil {
		return 0, err
	}

	intVal, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid integer value for setting '%s': %w", key, err)
	}

	return intVal, nil
}

// GetFloat retrieves a float setting value
func (s *settingsService) getFloat(key string) (float64, error) {
	val, err := s.getFromCacheOrDB(key)
	if err != nil {
		return 0, err
	}

	floatVal, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float value for setting '%s': %w", key, err)
	}

	return floatVal, nil
}

// GetBool retrieves a boolean setting value
func (s *settingsService) getBool(key string) (bool, error) {
	val, err := s.getFromCacheOrDB(key)
	if err != nil {
		return false, err
	}

	boolVal, err := strconv.ParseBool(val)
	if err != nil {
		return false, fmt.Errorf("invalid boolean value for setting '%s': %w", key, err)
	}

	return boolVal, nil
}

// Pricing methods

func (s *settingsService) GetPlanBasePrice(plan string) (int, error) {
	var key string
	switch plan {
	case "free":
		key = "free_plan_base_price"
	case "eco":
		key = "eco_plan_base_price"
	case "pro":
		key = "pro_plan_base_price"
	default:
		return 0, fmt.Errorf("invalid plan: %s", plan)
	}
	return s.getInt(key)
}

func (s *settingsService) GetPlanFreeMinutes(plan string) (int, error) {
	var key string
	switch plan {
	case "free":
		key = "free_plan_free_minutes"
	case "eco":
		key = "eco_plan_free_minutes"
	case "pro":
		key = "pro_plan_free_minutes"
	default:
		return 0, fmt.Errorf("invalid plan: %s", plan)
	}
	return s.getInt(key)
}

func (s *settingsService) GetPlanRuntimePricePerMinute(plan string) (float64, error) {
	var key string
	switch plan {
	case "free":
		key = "free_plan_runtime_price_per_minute"
	case "eco":
		key = "eco_plan_runtime_price_per_minute"
	case "pro":
		key = "pro_plan_runtime_price_per_minute"
	default:
		return 0, fmt.Errorf("invalid plan: %s", plan)
	}
	return s.getFloat(key)
}

// Resource pricing - Eco plan

func (s *settingsService) GetEcoCPUPricePerCorePerMinute() (float64, error) {
	return s.getFloat("eco_cpu_price_per_core_per_minute")
}

func (s *settingsService) GetEcoMemoryPricePerGBPerMinute() (float64, error) {
	return s.getFloat("eco_memory_price_per_gb_per_minute")
}

func (s *settingsService) GetEcoDiskPricePerGBPerMonth() (int, error) {
	return s.getInt("eco_disk_price_per_gb_per_month")
}

// Resource pricing - Pro plan

func (s *settingsService) GetProCPUPricePerCorePerMonth() (int, error) {
	return s.getInt("pro_cpu_price_per_core_per_month")
}

func (s *settingsService) GetProMemoryPricePerGBPerMonth() (int, error) {
	return s.getInt("pro_memory_price_per_gb_per_month")
}

func (s *settingsService) GetProDiskPricePerGBPerMonth() (int, error) {
	return s.getInt("pro_disk_price_per_gb_per_month")
}

// Free plan limits

func (s *settingsService) GetFreePlanCPULimit() (int, error) {
	return s.getInt("free_plan_cpu_limit")
}

func (s *settingsService) GetFreePlanMemoryLimit() (int, error) {
	return s.getInt("free_plan_memory_limit")
}

func (s *settingsService) GetFreePlanDiskLimit() (int, error) {
	return s.getInt("free_plan_disk_limit")
}

func (s *settingsService) GetFreePlanMaxProjects() (int, error) {
	return s.getInt("free_plan_max_projects")
}

// Project count limits

func (s *settingsService) GetMaxProjectsPerUser() (int, error) {
	return s.getInt("max_projects_per_user")
}

// Beta tier limits

func (s *settingsService) IsBetaTierEnabled() (bool, error) {
	return s.getBool("beta_tier_enabled")
}

func (s *settingsService) GetBetaTierCPULimit() (int, error) {
	return s.getInt("beta_tier_cpu_limit")
}

func (s *settingsService) GetBetaTierMemoryLimit() (int, error) {
	return s.getInt("beta_tier_memory_limit")
}

func (s *settingsService) GetBetaTierDiskLimit() (int, error) {
	return s.getInt("beta_tier_disk_limit")
}

// Direct access methods for handlers

func (s *settingsService) GetByKey(key string) (*Setting, error) {
	return s.repo.GetByKey(context.Background(), key)
}

func (s *settingsService) GetByCategory(category string) ([]*Setting, error) {
	return s.repo.GetByCategory(context.Background(), category)
}

func (s *settingsService) GetAll() ([]*Setting, error) {
	return s.repo.GetAll(context.Background())
}

func (s *settingsService) UpdateSetting(key, value string, updatedBy *int) error {
	// Update in database
	err := s.repo.Update(context.Background(), key, value, updatedBy)
	if err != nil {
		return err
	}

	// Invalidate cache for this key
	s.InvalidateCache(key)

	return nil
}
