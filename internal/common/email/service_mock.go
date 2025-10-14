package email

import "github.com/stretchr/testify/mock"

type MockService struct {
	mock.Mock
}

func (m *MockService) SendVerificationEmail(email, username, token string) error {
	args := m.Called(email, username, token)
	return args.Error(0)
}

func (m *MockService) SendPasswordResetEmail(email, username, token string) error {
	args := m.Called(email, username, token)
	return args.Error(0)
}
