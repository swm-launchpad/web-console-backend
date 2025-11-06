package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
)

func TestGetContainerSlugsByProjectIDUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 컨테이너 slug 조회", func(t *testing.T) {
		mockRepo := new(repository.MockContainerRepository)
		testLogger := logger.NewForTest()
		uc := NewGetContainerSlugsByProjectIDUseCase(mockRepo, testLogger)

		expectedSlugs := []string{"c20251106123456abcd1234", "c20251106789012efgh5678"}
		mockRepo.On("FindAllSlugsByProjectIDIncludingDeleted", mock.Anything, uint(1)).Return(expectedSlugs, nil)

		input := GetContainerSlugsByProjectIDInput{
			ProjectID: 1,
		}

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, expectedSlugs, output.ContainerSlugs)
		mockRepo.AssertExpectations(t)
	})

	t.Run("성공: 컨테이너가 없는 경우", func(t *testing.T) {
		mockRepo := new(repository.MockContainerRepository)
		testLogger := logger.NewForTest()
		uc := NewGetContainerSlugsByProjectIDUseCase(mockRepo, testLogger)

		mockRepo.On("FindAllSlugsByProjectIDIncludingDeleted", mock.Anything, uint(1)).Return([]string{}, nil)

		input := GetContainerSlugsByProjectIDInput{
			ProjectID: 1,
		}

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Empty(t, output.ContainerSlugs)
		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 저장소 에러", func(t *testing.T) {
		mockRepo := new(repository.MockContainerRepository)
		testLogger := logger.NewForTest()
		uc := NewGetContainerSlugsByProjectIDUseCase(mockRepo, testLogger)

		dbError := errors.New("database error")
		mockRepo.On("FindAllSlugsByProjectIDIncludingDeleted", mock.Anything, uint(1)).Return(nil, dbError)

		input := GetContainerSlugsByProjectIDInput{
			ProjectID: 1,
		}

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, dbError, err)
		mockRepo.AssertExpectations(t)
	})
}
