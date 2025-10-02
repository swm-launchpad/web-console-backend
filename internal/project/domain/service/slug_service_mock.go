package service

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
)

type MockSlugService struct {
	mock.Mock
}

func (m *MockSlugService) EnsureUniqueSlug(ctx context.Context, slug value.ProjectSlug) error {
	args := m.Called(ctx, slug)
	return args.Error(0)
}

func (m *MockSlugService) GenerateSlugFromName(ctx context.Context, name string) (value.ProjectSlug, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(value.ProjectSlug), args.Error(1)
}
