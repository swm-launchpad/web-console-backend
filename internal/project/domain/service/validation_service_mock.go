package service

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
)

// MockValidationService is a mock implementation of ValidationService for testing
type MockValidationService struct {
	ValidateFreeResourcesFunc      func(plan value.Plan, limits value.ResourceLimits) error
	ValidateFreeTierLimitsFunc     func(plan value.Plan, limits value.ResourceLimits) error
	ValidateFreePlanLimitFunc      func(ctx context.Context, userID uint, plan value.Plan) error
	ValidateMaxProjectsPerUserFunc func(ctx context.Context, userID uint) error
}

// ValidateFreeResources mocks the ValidateFreeResources method
func (m *MockValidationService) ValidateFreeResources(plan value.Plan, limits value.ResourceLimits) error {
	if m.ValidateFreeResourcesFunc != nil {
		return m.ValidateFreeResourcesFunc(plan, limits)
	}
	return nil
}

// ValidateFreeTierLimits mocks the ValidateFreeTierLimits method
func (m *MockValidationService) ValidateFreeTierLimits(plan value.Plan, limits value.ResourceLimits) error {
	if m.ValidateFreeTierLimitsFunc != nil {
		return m.ValidateFreeTierLimitsFunc(plan, limits)
	}
	return nil
}

// ValidateFreePlanLimit mocks the ValidateFreePlanLimit method
func (m *MockValidationService) ValidateFreePlanLimit(ctx context.Context, userID uint, plan value.Plan) error {
	if m.ValidateFreePlanLimitFunc != nil {
		return m.ValidateFreePlanLimitFunc(ctx, userID, plan)
	}
	return nil
}

// ValidateMaxProjectsPerUser mocks the ValidateMaxProjectsPerUser method
func (m *MockValidationService) ValidateMaxProjectsPerUser(ctx context.Context, userID uint) error {
	if m.ValidateMaxProjectsPerUserFunc != nil {
		return m.ValidateMaxProjectsPerUserFunc(ctx, userID)
	}
	return nil
}
