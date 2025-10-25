package infrastructure

import (
	"context"

	"github.com/stretchr/testify/mock"
	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/template"
)

// MockTemplateRepository is a mock implementation of TemplateRepository for testing
type MockTemplateRepository struct {
	mock.Mock
}

func (m *MockTemplateRepository) FindAll(ctx context.Context) ([]*model.Template, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Template), args.Error(1)
}

func (m *MockTemplateRepository) FindByID(ctx context.Context, templateID uint) (*model.Template, error) {
	args := m.Called(ctx, templateID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Template), args.Error(1)
}

func (m *MockTemplateRepository) FindActiveTemplates(ctx context.Context) ([]*model.Template, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Template), args.Error(1)
}

func (m *MockTemplateRepository) ExistsByID(ctx context.Context, templateID uint) (bool, error) {
	args := m.Called(ctx, templateID)
	return args.Bool(0), args.Error(1)
}
