package service

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
)

type MockSlugService struct {
	mock.Mock
}

func (m *MockSlugService) EnsureUniqueSlug(ctx context.Context, projectID uint, slug value.ContainerSlug) error {
	args := m.Called(ctx, projectID, slug)
	return args.Error(0)
}

func (m *MockSlugService) GenerateSlugFromName(ctx context.Context, projectID uint, name string) (value.ContainerSlug, error) {
	args := m.Called(ctx, projectID, name)
	return args.Get(0).(value.ContainerSlug), args.Error(1)
}
