package infrastructure

import (
	"github.com/stretchr/testify/mock"
)

// MockStateValidator is a mock implementation of StateValidator for testing
type MockStateValidator struct {
	mock.Mock
}

func (m *MockStateValidator) GenerateState(userID uint) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockStateValidator) ValidateState(state string) (uint, error) {
	args := m.Called(state)
	return uint(args.Int(0)), args.Error(1)
}
