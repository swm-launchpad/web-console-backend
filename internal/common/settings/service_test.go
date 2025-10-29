package settings

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockSettingsRepository is a mock implementation of SettingsRepository
type MockSettingsRepository struct {
	mock.Mock
}

func (m *MockSettingsRepository) GetByKey(ctx context.Context, key string) (*Setting, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Setting), args.Error(1)
}

func (m *MockSettingsRepository) GetByCategory(ctx context.Context, category string) ([]*Setting, error) {
	args := m.Called(ctx, category)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Setting), args.Error(1)
}

func (m *MockSettingsRepository) GetAll(ctx context.Context) ([]*Setting, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Setting), args.Error(1)
}

func (m *MockSettingsRepository) Update(ctx context.Context, key, value string, updatedBy *int) error {
	args := m.Called(ctx, key, value, updatedBy)
	return args.Error(0)
}

func TestSettingsService_GetPlanBasePrice(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: Free plan 기본 가격 조회", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		service := NewSettingsService(mockRepo)

		mockRepo.On("GetByKey", ctx, "free_plan_base_price").Return(&Setting{
			Key:       "free_plan_base_price",
			Value:     "0",
			ValueType: "int",
		}, nil)

		price, err := service.GetPlanBasePrice("free")

		require.NoError(t, err)
		assert.Equal(t, 0, price)
		mockRepo.AssertExpectations(t)
	})

	t.Run("성공: Eco plan 기본 가격 조회", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		service := NewSettingsService(mockRepo)

		mockRepo.On("GetByKey", ctx, "eco_plan_base_price").Return(&Setting{
			Key:       "eco_plan_base_price",
			Value:     "1100",
			ValueType: "int",
		}, nil)

		price, err := service.GetPlanBasePrice("eco")

		require.NoError(t, err)
		assert.Equal(t, 1100, price)
		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 잘못된 plan 타입", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		service := NewSettingsService(mockRepo)

		price, err := service.GetPlanBasePrice("invalid")

		assert.Error(t, err)
		assert.Equal(t, 0, price)
		assert.Contains(t, err.Error(), "invalid plan")
	})

	t.Run("실패: 설정을 찾을 수 없음", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		service := NewSettingsService(mockRepo)

		mockRepo.On("GetByKey", ctx, "free_plan_base_price").Return(nil, errors.New("not found"))

		price, err := service.GetPlanBasePrice("free")

		assert.Error(t, err)
		assert.Equal(t, 0, price)
		mockRepo.AssertExpectations(t)
	})
}

func TestSettingsService_Caching(t *testing.T) {
	ctx := context.Background()

	t.Run("캐시 히트: 동일한 설정 두 번 조회 시 DB 호출 한 번만", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		service := NewSettingsService(mockRepo)

		// 첫 번째 호출만 DB에서 가져옴
		mockRepo.On("GetByKey", ctx, "eco_plan_base_price").Return(&Setting{
			Key:       "eco_plan_base_price",
			Value:     "1100",
			ValueType: "int",
		}, nil).Once()

		// 첫 번째 호출 - DB에서 가져옴
		price1, err1 := service.GetPlanBasePrice("eco")
		require.NoError(t, err1)
		assert.Equal(t, 1100, price1)

		// 두 번째 호출 - 캐시에서 가져옴 (DB 호출 없음)
		price2, err2 := service.GetPlanBasePrice("eco")
		require.NoError(t, err2)
		assert.Equal(t, 1100, price2)

		mockRepo.AssertExpectations(t)
	})

	t.Run("캐시 만료: TTL 이후 다시 DB 조회", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		// 테스트용으로 짧은 TTL 사용
		service := &settingsService{
			repo: mockRepo,
			ttl:  10 * time.Millisecond,
		}

		mockRepo.On("GetByKey", ctx, "eco_plan_base_price").Return(&Setting{
			Key:       "eco_plan_base_price",
			Value:     "1100",
			ValueType: "int",
		}, nil).Twice() // 두 번 호출 예상

		// 첫 번째 호출
		price1, err1 := service.GetPlanBasePrice("eco")
		require.NoError(t, err1)
		assert.Equal(t, 1100, price1)

		// TTL 만료 대기
		time.Sleep(20 * time.Millisecond)

		// 두 번째 호출 - 캐시 만료로 DB 재조회
		price2, err2 := service.GetPlanBasePrice("eco")
		require.NoError(t, err2)
		assert.Equal(t, 1100, price2)

		mockRepo.AssertExpectations(t)
	})
}

func TestSettingsService_GetFreePlanCPULimit(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: Free plan CPU 제한 조회", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		service := NewSettingsService(mockRepo)

		mockRepo.On("GetByKey", ctx, "free_plan_cpu_limit").Return(&Setting{
			Key:       "free_plan_cpu_limit",
			Value:     "500",
			ValueType: "int",
		}, nil)

		limit, err := service.GetFreePlanCPULimit()

		require.NoError(t, err)
		assert.Equal(t, 500, limit)
		mockRepo.AssertExpectations(t)
	})
}

func TestSettingsService_IsBetaTierEnabled(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: Beta tier 활성화 상태 조회 (true)", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		service := NewSettingsService(mockRepo)

		mockRepo.On("GetByKey", ctx, "beta_tier_enabled").Return(&Setting{
			Key:       "beta_tier_enabled",
			Value:     "true",
			ValueType: "boolean",
		}, nil)

		enabled, err := service.IsBetaTierEnabled()

		require.NoError(t, err)
		assert.True(t, enabled)
		mockRepo.AssertExpectations(t)
	})

	t.Run("성공: Beta tier 비활성화 상태 조회 (false)", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		service := NewSettingsService(mockRepo)

		mockRepo.On("GetByKey", ctx, "beta_tier_enabled").Return(&Setting{
			Key:       "beta_tier_enabled",
			Value:     "false",
			ValueType: "boolean",
		}, nil)

		enabled, err := service.IsBetaTierEnabled()

		require.NoError(t, err)
		assert.False(t, enabled)
		mockRepo.AssertExpectations(t)
	})
}

func TestSettingsService_GetPlanRuntimePricePerMinute(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: Eco plan 분당 실행 비용 조회", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		service := NewSettingsService(mockRepo)

		mockRepo.On("GetByKey", ctx, "eco_plan_runtime_price_per_minute").Return(&Setting{
			Key:       "eco_plan_runtime_price_per_minute",
			Value:     "3.3",
			ValueType: "float",
		}, nil)

		price, err := service.GetPlanRuntimePricePerMinute("eco")

		require.NoError(t, err)
		assert.Equal(t, 3.3, price)
		mockRepo.AssertExpectations(t)
	})
}

func TestSettingsService_InvalidateCache(t *testing.T) {
	ctx := context.Background()

	t.Run("캐시 무효화 후 DB 재조회", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		service := NewSettingsService(mockRepo)

		// 첫 번째 호출
		mockRepo.On("GetByKey", ctx, "eco_plan_base_price").Return(&Setting{
			Key:   "eco_plan_base_price",
			Value: "1100",
		}, nil).Once()

		price1, err1 := service.GetPlanBasePrice("eco")
		require.NoError(t, err1)
		assert.Equal(t, 1100, price1)

		// 캐시 무효화
		service.InvalidateCache("eco_plan_base_price")

		// 두 번째 호출 - DB 재조회
		mockRepo.On("GetByKey", ctx, "eco_plan_base_price").Return(&Setting{
			Key:   "eco_plan_base_price",
			Value: "1200", // 값이 변경됨
		}, nil).Once()

		price2, err2 := service.GetPlanBasePrice("eco")
		require.NoError(t, err2)
		assert.Equal(t, 1200, price2)

		mockRepo.AssertExpectations(t)
	})
}

func TestSettingsService_GetByCategory(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: pricing 카테고리 조회", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		service := NewSettingsService(mockRepo)

		expectedSettings := []*Setting{
			{Key: "free_plan_base_price", Value: "0", ValueType: "int", Category: "pricing"},
			{Key: "eco_plan_base_price", Value: "1100", ValueType: "int", Category: "pricing"},
		}

		mockRepo.On("GetByCategory", ctx, "pricing").Return(expectedSettings, nil)

		settings, err := service.GetByCategory("pricing")

		require.NoError(t, err)
		assert.Len(t, settings, 2)
		assert.Equal(t, "free_plan_base_price", settings[0].Key)
		mockRepo.AssertExpectations(t)
	})
}

func TestSettingsService_TypeParsing(t *testing.T) {
	ctx := context.Background()

	t.Run("실패: 잘못된 정수 형식", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		service := NewSettingsService(mockRepo)

		mockRepo.On("GetByKey", ctx, "free_plan_base_price").Return(&Setting{
			Key:       "free_plan_base_price",
			Value:     "not_a_number",
			ValueType: "int",
		}, nil)

		price, err := service.GetPlanBasePrice("free")

		assert.Error(t, err)
		assert.Equal(t, 0, price)
		assert.Contains(t, err.Error(), "invalid syntax")
	})

	t.Run("실패: 잘못된 실수 형식", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		service := NewSettingsService(mockRepo)

		mockRepo.On("GetByKey", ctx, "eco_plan_runtime_price_per_minute").Return(&Setting{
			Key:       "eco_plan_runtime_price_per_minute",
			Value:     "not_a_float",
			ValueType: "float",
		}, nil)

		price, err := service.GetPlanRuntimePricePerMinute("eco")

		assert.Error(t, err)
		assert.Equal(t, 0.0, price)
		assert.Contains(t, err.Error(), "invalid syntax")
	})

	t.Run("실패: 잘못된 불리언 형식", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		service := NewSettingsService(mockRepo)

		mockRepo.On("GetByKey", ctx, "beta_tier_enabled").Return(&Setting{
			Key:       "beta_tier_enabled",
			Value:     "not_a_bool",
			ValueType: "boolean",
		}, nil)

		enabled, err := service.IsBetaTierEnabled()

		assert.Error(t, err)
		assert.False(t, enabled)
		assert.Contains(t, err.Error(), "invalid syntax")
	})
}
