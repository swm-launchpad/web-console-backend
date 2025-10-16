package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
)

func TestListContainersUseCase_Execute_Success(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	useCase := NewListContainersUseCase(mockRepo, mockPermSvc)

	ctx := context.Background()
	projectID := uint(10)
	userID := uint(100)

	input := ListContainersInput{
		ProjectID: projectID,
		UserID:    userID,
	}

	mockContainers := []*model.Container{
		createMockContainer(1, projectID),
		createMockContainer(2, projectID),
		createMockContainer(3, projectID),
	}

	mockPermSvc.On("CanUserCreateContainer", ctx, userID, projectID).Return(nil)
	mockRepo.On("FindByProjectID", ctx, projectID).Return(mockContainers, nil)
	mockRepo.On("CountByProjectID", ctx, projectID).Return(int64(3), nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.Containers, 3)
	assert.Equal(t, int64(3), output.Total)

	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestListContainersUseCase_Execute_PermissionDenied(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	useCase := NewListContainersUseCase(mockRepo, mockPermSvc)

	ctx := context.Background()
	projectID := uint(10)
	userID := uint(100)

	input := ListContainersInput{
		ProjectID: projectID,
		UserID:    userID,
	}

	mockPermSvc.On("CanUserCreateContainer", ctx, userID, projectID).Return(assert.AnError)

	output, err := useCase.Execute(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, output)

	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "FindByProjectID")
	mockRepo.AssertNotCalled(t, "CountByProjectID")
}

func TestListContainersUseCase_Execute_EmptyList(t *testing.T) {
	mockRepo := new(infrastructure.MockContainerRepository)
	mockPermSvc := new(infrastructure.MockPermissionService)
	useCase := NewListContainersUseCase(mockRepo, mockPermSvc)

	ctx := context.Background()
	projectID := uint(10)
	userID := uint(100)

	input := ListContainersInput{
		ProjectID: projectID,
		UserID:    userID,
	}

	mockContainers := []*model.Container{}

	mockPermSvc.On("CanUserCreateContainer", ctx, userID, projectID).Return(nil)
	mockRepo.On("FindByProjectID", ctx, projectID).Return(mockContainers, nil)
	mockRepo.On("CountByProjectID", ctx, projectID).Return(int64(0), nil)

	output, err := useCase.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.Len(t, output.Containers, 0)
	assert.Equal(t, int64(0), output.Total)

	mockPermSvc.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}
