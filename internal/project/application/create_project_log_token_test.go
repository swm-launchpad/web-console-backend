package application

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	projectmodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
)

func TestCreateProjectLogTokenUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	jwtSecret := "test-secret-key-12345"
	log := logger.NewForTest()

	useCase := NewCreateProjectLogTokenUseCase(
		mockProjectRepo,
		jwtSecret,
		log,
	)

	input := CreateProjectLogTokenInput{
		ProjectID: 10,
		UserID:    1,
	}

	// Create mock project
	mockProject := createTestProjectForLogTests(10, 1, "Test Project", "p2025011812000012345678")

	// Mock FindByID - returns project
	mockProjectRepo.On("FindByID", ctx, uint(10)).Return(mockProject, nil)

	// Execute
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.NotEmpty(t, output.Token)

	// Verify token can be validated
	token, parseErr := jwt.Parse(output.Token, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	assert.NoError(t, parseErr)
	assert.True(t, token.Valid)

	// Verify claims
	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, float64(1), claims["user_id"])
	assert.Equal(t, float64(10), claims["project_id"])

	// Verify expiration (15 minutes)
	exp := int64(claims["exp"].(float64))
	expTime := time.Unix(exp, 0)
	assert.True(t, expTime.After(time.Now()))
	assert.True(t, expTime.Before(time.Now().Add(16*time.Minute)))

	mockProjectRepo.AssertExpectations(t)
}

func TestCreateProjectLogTokenUseCase_Execute_ProjectNotFound(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	jwtSecret := "test-secret-key-12345"
	log := logger.NewForTest()

	useCase := NewCreateProjectLogTokenUseCase(
		mockProjectRepo,
		jwtSecret,
		log,
	)

	input := CreateProjectLogTokenInput{
		ProjectID: 999, // Non-existent project
		UserID:    1,
	}

	// Mock FindByID - project not found
	mockProjectRepo.On("FindByID", ctx, uint(999)).
		Return(nil, projecterrors.ErrProjectNotFound)

	// Execute
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, projecterrors.ErrProjectNotFound, err)
	assert.Nil(t, output)

	mockProjectRepo.AssertExpectations(t)
}

func TestCreateProjectLogTokenUseCase_Execute_TokenExpiration(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	jwtSecret := "test-secret-key-12345"
	log := logger.NewForTest()

	useCase := NewCreateProjectLogTokenUseCase(
		mockProjectRepo,
		jwtSecret,
		log,
	)

	input := CreateProjectLogTokenInput{
		ProjectID: 10,
		UserID:    1,
	}

	// Create mock project
	mockProject := createTestProjectForLogTests(10, 1, "Test Project", "p2025011812000012345678")

	mockProjectRepo.On("FindByID", ctx, uint(10)).Return(mockProject, nil)

	// Execute multiple times and verify each token has proper expiration
	for i := 0; i < 3; i++ {
		output, err := useCase.Execute(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, output)

		// Parse token and verify expiration
		token, parseErr := jwt.Parse(output.Token, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		assert.NoError(t, parseErr)

		claims := token.Claims.(jwt.MapClaims)
		exp := int64(claims["exp"].(float64))
		expTime := time.Unix(exp, 0)

		// Verify expiration is approximately 15 minutes from now (±1 minute for test execution time)
		expectedExpiration := time.Now().Add(15 * time.Minute)
		timeDiff := expTime.Sub(expectedExpiration).Abs()
		assert.Less(t, timeDiff, 1*time.Minute, "Expiration time should be within 1 minute of expected")
	}

	mockProjectRepo.AssertExpectations(t)
}

// createTestProjectForLogTests creates a test project with minimal configuration for log tests
func createTestProjectForLogTests(id uint, ownerID uint, name string, slugStr string) *projectmodel.Project {
	projectSlug, err := value.NewProjectSlug(slugStr)
	if err != nil {
		panic(err)
	}
	limits, err := value.NewResourceLimits(500, 512, 1024, 128)
	if err != nil {
		panic(err)
	}
	project, err := projectmodel.NewProject(name, *projectSlug, ownerID, *limits, nil)
	if err != nil {
		panic(err)
	}
	project.SetProjectID(id)
	return project
}
