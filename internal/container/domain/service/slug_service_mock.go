package service

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
)

type MockSlugService struct {
	mock.Mock
}

func (m *MockSlugService) EnsureUniqueSlug(ctx context.Context, slug value.ContainerSlug) error {
	args := m.Called(ctx, slug)
	return args.Error(0)
}

func (m *MockSlugService) GenerateSlug(ctx context.Context) (value.ContainerSlug, error) {
	args := m.Called(ctx)
	return args.Get(0).(value.ContainerSlug), args.Error(1)
}
