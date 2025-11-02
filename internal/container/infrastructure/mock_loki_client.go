package infrastructure

import (
	"context"
	"io"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockLokiClient is a mock implementation of LokiClient for testing
type MockLokiClient struct {
	mock.Mock
}

// StreamPipelineRunLogs mocks the StreamPipelineRunLogs method
func (m *MockLokiClient) StreamPipelineRunLogs(ctx context.Context, pipelineRunName string, excludeTasks []string) (io.ReadCloser, error) {
	args := m.Called(ctx, pipelineRunName, excludeTasks)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// QueryPipelineRunLogsHTTP mocks the QueryPipelineRunLogsHTTP method
func (m *MockLokiClient) QueryPipelineRunLogsHTTP(ctx context.Context, pipelineRunName string, excludeTasks []string, startTime, endTime time.Time) (io.ReadCloser, error) {
	args := m.Called(ctx, pipelineRunName, excludeTasks, startTime, endTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}
