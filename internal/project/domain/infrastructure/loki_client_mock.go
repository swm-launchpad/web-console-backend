package infrastructure

import (
	"context"
	"io"
	"time"

	"github.com/stretchr/testify/mock"
)

type MockLokiClient struct {
	mock.Mock
}

func (m *MockLokiClient) StreamApplicationLogs(ctx context.Context, projectSlug string, since time.Time) (io.ReadCloser, error) {
	args := m.Called(ctx, projectSlug, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockLokiClient) QueryApplicationLogsHistory(ctx context.Context, projectSlug string, before time.Time, limit int) ([]ApplicationLogEntry, error) {
	args := m.Called(ctx, projectSlug, before, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ApplicationLogEntry), args.Error(1)
}

func (m *MockLokiClient) QueryApplicationLogsAfter(ctx context.Context, projectSlug string, after time.Time, limit int) ([]ApplicationLogEntry, error) {
	args := m.Called(ctx, projectSlug, after, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ApplicationLogEntry), args.Error(1)
}

func (m *MockLokiClient) QueryApplicationLogsHistoryRaw(ctx context.Context, projectSlug string, before time.Time, limit int) (io.ReadCloser, error) {
	args := m.Called(ctx, projectSlug, before, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockLokiClient) QueryApplicationLogsAfterRaw(ctx context.Context, projectSlug string, after time.Time, limit int) (io.ReadCloser, error) {
	args := m.Called(ctx, projectSlug, after, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}
