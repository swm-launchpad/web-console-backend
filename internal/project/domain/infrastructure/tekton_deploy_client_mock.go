package infrastructure

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

type MockTektonClient struct {
	mock.Mock
}

func (m *MockTektonClient) TriggerDeploy(ctx context.Context, request *dto.TektonDeployRequest) (*dto.TektonDeployResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TektonDeployResponse), args.Error(1)
}
