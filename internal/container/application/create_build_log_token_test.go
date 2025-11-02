package application

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
)

func TestCreateBuildLogTokenUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockPermissionService := new(infrastructure.MockPermissionService)
	jwtUtil := jwt.NewJWTUtil("test-secret")
	log := logger.NewForTest()

	useCase := NewCreateBuildLogTokenUseCase(
		mockPermissionService,
		jwtUtil,
		log,
	)

	input := CreateBuildLogTokenInput{
		UserID:      1,
		ContainerID: 10,
	}

	// Mock permission check - user has access
	mockPermissionService.On("CanUserAccessContainer", ctx, uint(1), uint(10)).Return(nil)

	// Execute
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.NotEmpty(t, output.Token)
	assert.True(t, output.ExpiresAt.After(time.Now()))
	assert.True(t, output.ExpiresAt.Before(time.Now().Add(31*time.Minute)))

	// Verify token can be validated
	claims, err := jwtUtil.ValidateBuildLogToken(ctx, output.Token)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), claims.UserID)
	assert.Equal(t, uint(10), claims.ContainerID)

	mockPermissionService.AssertExpectations(t)
}

func TestCreateBuildLogTokenUseCase_Execute_PermissionDenied(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockPermissionService := new(infrastructure.MockPermissionService)
	jwtUtil := jwt.NewJWTUtil("test-secret")
	log := logger.NewForTest()

	useCase := NewCreateBuildLogTokenUseCase(
		mockPermissionService,
		jwtUtil,
		log,
	)

	input := CreateBuildLogTokenInput{
		UserID:      1,
		ContainerID: 10,
	}

	// Mock permission check - user does NOT have access
	mockPermissionService.On("CanUserAccessContainer", ctx, uint(1), uint(10)).
		Return(containererrors.ErrPermissionDenied)

	// Execute
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrPermissionDenied, err)
	assert.Nil(t, output)

	mockPermissionService.AssertExpectations(t)
}

func TestCreateBuildLogTokenUseCase_Execute_ContainerNotFound(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockPermissionService := new(infrastructure.MockPermissionService)
	jwtUtil := jwt.NewJWTUtil("test-secret")
	log := logger.NewForTest()

	useCase := NewCreateBuildLogTokenUseCase(
		mockPermissionService,
		jwtUtil,
		log,
	)

	input := CreateBuildLogTokenInput{
		UserID:      1,
		ContainerID: 999, // Non-existent container
	}

	// Mock permission check - container not found
	mockPermissionService.On("CanUserAccessContainer", ctx, uint(1), uint(999)).
		Return(containererrors.ErrContainerNotFound)

	// Execute
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, containererrors.ErrContainerNotFound, err)
	assert.Nil(t, output)

	mockPermissionService.AssertExpectations(t)
}

func TestCreateBuildLogTokenUseCase_Execute_TokenExpiration(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockPermissionService := new(infrastructure.MockPermissionService)
	jwtUtil := jwt.NewJWTUtil("test-secret")
	log := logger.NewForTest()

	useCase := NewCreateBuildLogTokenUseCase(
		mockPermissionService,
		jwtUtil,
		log,
	)

	input := CreateBuildLogTokenInput{
		UserID:      1,
		ContainerID: 10,
	}

	mockPermissionService.On("CanUserAccessContainer", ctx, mock.Anything, mock.Anything).Return(nil)

	// Execute multiple times and verify each token has proper expiration
	for i := 0; i < 3; i++ {
		output, err := useCase.Execute(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, output)

		// Verify expiration is approximately 30 minutes from now (±1 minute for test execution time)
		expectedExpiration := time.Now().Add(30 * time.Minute)
		timeDiff := output.ExpiresAt.Sub(expectedExpiration).Abs()
		assert.Less(t, timeDiff, 1*time.Minute, "Expiration time should be within 1 minute of expected")
	}

	mockPermissionService.AssertExpectations(t)
}
