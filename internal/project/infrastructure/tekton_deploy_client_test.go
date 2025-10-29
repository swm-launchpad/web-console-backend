package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

func TestNewTektonDeployClient(t *testing.T) {
	tests := []struct {
		name           string
		deployURL      string
		authHeader     string
		expectedError  error
		shouldSetupEnv bool
	}{
		{
			name:           "성공: 유효한 환경변수로 클라이언트 생성",
			deployURL:      "https://test-tekton-api.example.com/deploy",
			authHeader:     "Basic dGVzdDp0ZXN0",
			expectedError:  nil,
			shouldSetupEnv: true,
		},
		{
			name:           "실패: TEKTON_DEPLOY_URL 누락",
			deployURL:      "",
			authHeader:     "Basic dGVzdDp0ZXN0",
			expectedError:  projecterrors.ErrTektonUnavailable,
			shouldSetupEnv: true,
		},
		{
			name:           "실패: TEKTON_API_AUTH 누락",
			deployURL:      "https://test-tekton-api.example.com/deploy",
			authHeader:     "",
			expectedError:  projecterrors.ErrTektonUnavailable,
			shouldSetupEnv: true,
		},
		{
			name:           "실패: 모든 환경변수 누락",
			deployURL:      "",
			authHeader:     "",
			expectedError:  projecterrors.ErrTektonUnavailable,
			shouldSetupEnv: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSetupEnv {
				// Backup original env vars
				originalDeployURL := os.Getenv("TEKTON_DEPLOY_URL")
				originalAuthHeader := os.Getenv("TEKTON_API_AUTH")

				// Set test env vars
				if tt.deployURL != "" {
					require.NoError(t, os.Setenv("TEKTON_DEPLOY_URL", tt.deployURL))
				} else {
					require.NoError(t, os.Unsetenv("TEKTON_DEPLOY_URL"))
				}

				if tt.authHeader != "" {
					require.NoError(t, os.Setenv("TEKTON_API_AUTH", tt.authHeader))
				} else {
					require.NoError(t, os.Unsetenv("TEKTON_API_AUTH"))
				}

				// Restore env vars after test
				defer func() {
					if originalDeployURL != "" {
						_ = os.Setenv("TEKTON_DEPLOY_URL", originalDeployURL)
					} else {
						_ = os.Unsetenv("TEKTON_DEPLOY_URL")
					}

					if originalAuthHeader != "" {
						_ = os.Setenv("TEKTON_API_AUTH", originalAuthHeader)
					} else {
						_ = os.Unsetenv("TEKTON_API_AUTH")
					}
				}()
			}

			client, err := NewTektonDeployClient(logger.NewForTest())

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestTektonClient_TriggerDeploy(t *testing.T) {
	tests := []struct {
		name           string
		request        *dto.TektonDeployRequest
		mockStatusCode int
		mockResponse   interface{}
		expectedError  error
		checkResponse  func(t *testing.T, response *dto.TektonDeployResponse)
	}{
		{
			name: "성공: 정상 배포 요청 (202 Accepted)",
			request: &dto.TektonDeployRequest{
				DeploymentConfigJSON: dto.DeploymentConfig{
					ProjectID:    "test-project-id",
					ServiceName:  "test-service",
					Namespace:    "application",
					StableWindow: 180,
					ConfigMaps:   []dto.ConfigMapInfo{},
					Volumes:      []dto.VolumeInfo{},
					Containers:   []dto.TektonContainerInfo{},
				},
				DryRun: "false",
			},
			mockStatusCode: http.StatusAccepted,
			mockResponse: dto.TektonDeployResponse{
				EventListener:    "deploy-listener",
				EventListenerUID: "abc-123-def-456",
				EventID:          "event-123",
				Namespace:        "deploy-pipeline",
			},
			expectedError: nil,
			checkResponse: func(t *testing.T, response *dto.TektonDeployResponse) {
				assert.Equal(t, "deploy-listener", response.EventListener)
				assert.Equal(t, "abc-123-def-456", response.EventListenerUID)
				assert.Equal(t, "event-123", response.EventID)
				assert.Equal(t, "deploy-pipeline", response.Namespace)
			},
		},
		{
			name: "성공: dry_run 모드",
			request: &dto.TektonDeployRequest{
				DeploymentConfigJSON: dto.DeploymentConfig{
					ProjectID:    "test-project-id",
					ServiceName:  "test-service",
					Namespace:    "application",
					StableWindow: 180,
					ConfigMaps:   []dto.ConfigMapInfo{},
					Volumes:      []dto.VolumeInfo{},
					Containers:   []dto.TektonContainerInfo{},
				},
				DryRun: "true",
			},
			mockStatusCode: http.StatusAccepted,
			mockResponse: dto.TektonDeployResponse{
				EventListener: "deploy-listener",
				EventID:       "event-456",
			},
			expectedError: nil,
			checkResponse: func(t *testing.T, response *dto.TektonDeployResponse) {
				assert.Equal(t, "deploy-listener", response.EventListener)
				assert.Equal(t, "event-456", response.EventID)
			},
		},
		{
			name: "실패: 잘못된 요청 (400 Bad Request)",
			request: &dto.TektonDeployRequest{
				DeploymentConfigJSON: dto.DeploymentConfig{},
				DryRun:               "false",
			},
			mockStatusCode: http.StatusBadRequest,
			mockResponse:   "validation failed: missing required fields",
			expectedError:  projecterrors.ErrTektonDeploymentFailed,
			checkResponse:  nil,
		},
		{
			name: "실패: 인증 실패 (401 Unauthorized)",
			request: &dto.TektonDeployRequest{
				DeploymentConfigJSON: dto.DeploymentConfig{
					ProjectID:   "test-project-id",
					ServiceName: "test-service",
					Namespace:   "application",
				},
				DryRun: "false",
			},
			mockStatusCode: http.StatusUnauthorized,
			mockResponse:   "unauthorized",
			expectedError:  projecterrors.ErrTektonAuthenticationFailed,
			checkResponse:  nil,
		},
		{
			name: "실패: 서버 에러 (500 Internal Server Error)",
			request: &dto.TektonDeployRequest{
				DeploymentConfigJSON: dto.DeploymentConfig{
					ProjectID:   "test-project-id",
					ServiceName: "test-service",
					Namespace:   "application",
				},
				DryRun: "false",
			},
			mockStatusCode: http.StatusInternalServerError,
			mockResponse:   "internal server error",
			expectedError:  projecterrors.ErrTektonUnavailable,
			checkResponse:  nil,
		},
		{
			name: "실패: 잘못된 응답 형식",
			request: &dto.TektonDeployRequest{
				DeploymentConfigJSON: dto.DeploymentConfig{
					ProjectID:   "test-project-id",
					ServiceName: "test-service",
					Namespace:   "application",
				},
				DryRun: "false",
			},
			mockStatusCode: http.StatusAccepted,
			mockResponse:   "not a json response",
			expectedError:  projecterrors.ErrInvalidTektonResponse,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				assert.Equal(t, http.MethodPost, r.Method)

				// Verify headers
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "Basic test-auth", r.Header.Get("Authorization"))

				// Send mock response
				w.WriteHeader(tt.mockStatusCode)

				// Write response body
				switch v := tt.mockResponse.(type) {
				case string:
					_, _ = w.Write([]byte(v))
				case dto.TektonDeployResponse:
					_ = json.NewEncoder(w).Encode(v)
				default:
					_ = json.NewEncoder(w).Encode(v)
				}
			}))
			defer server.Close()

			// Create client with mock server URL
			client := &tektonClient{
				deployURL:  server.URL,
				authHeader: "Basic test-auth",
				httpClient: server.Client(),
				logger:     logger.NewForTest(),
			}

			// Execute test
			ctx := context.Background()
			response, err := client.TriggerDeploy(ctx, tt.request)

			// Verify error
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)

				if tt.checkResponse != nil {
					tt.checkResponse(t, response)
				}
			}
		})
	}
}

func TestTektonClient_TriggerDeploy_NetworkError(t *testing.T) {
	t.Run("실패: 네트워크 에러 (서버 연결 불가)", func(t *testing.T) {
		// Create client with invalid URL (no server listening)
		client := &tektonClient{
			deployURL:  "http://localhost:9999",
			authHeader: "Basic test-auth",
			httpClient: &http.Client{},
			logger:     logger.NewForTest(),
		}

		request := &dto.TektonDeployRequest{
			DeploymentConfigJSON: dto.DeploymentConfig{
				ProjectID:   "test-project-id",
				ServiceName: "test-service",
				Namespace:   "application",
			},
			DryRun: "false",
		}

		ctx := context.Background()
		response, err := client.TriggerDeploy(ctx, request)

		assert.Error(t, err)
		assert.ErrorIs(t, err, projecterrors.ErrTektonUnavailable)
		assert.Nil(t, response)
	})
}

func TestTektonClient_TriggerDeploy_ContextCancellation(t *testing.T) {
	t.Run("실패: Context 취소", func(t *testing.T) {
		// Create mock server that delays response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// This should not be reached because context is cancelled immediately
			w.WriteHeader(http.StatusAccepted)
		}))
		defer server.Close()

		client := &tektonClient{
			deployURL:  server.URL,
			authHeader: "Basic test-auth",
			httpClient: server.Client(),
			logger:     logger.NewForTest(),
		}

		request := &dto.TektonDeployRequest{
			DeploymentConfigJSON: dto.DeploymentConfig{
				ProjectID:   "test-project-id",
				ServiceName: "test-service",
				Namespace:   "application",
			},
			DryRun: "false",
		}

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		response, err := client.TriggerDeploy(ctx, request)

		assert.Error(t, err)
		assert.ErrorIs(t, err, projecterrors.ErrTektonUnavailable)
		assert.Nil(t, response)
	})
}

func TestTektonClient_TriggerDeploy_RequestValidation(t *testing.T) {
	t.Run("요청 본문 검증", func(t *testing.T) {
		var receivedRequest dto.TektonDeployRequest

		// Create mock server to capture request
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Decode and capture the request
			err := json.NewDecoder(r.Body).Decode(&receivedRequest)
			require.NoError(t, err)

			// Send success response
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(dto.TektonDeployResponse{
				EventListener: "deploy-listener",
				EventID:       "test-event",
			})
		}))
		defer server.Close()

		client := &tektonClient{
			deployURL:  server.URL,
			authHeader: "Basic test-auth",
			httpClient: server.Client(),
			logger:     logger.NewForTest(),
		}

		expectedRequest := &dto.TektonDeployRequest{
			DeploymentConfigJSON: dto.DeploymentConfig{
				ProjectID:    "my-project-123",
				ServiceName:  "my-service",
				Namespace:    "application",
				StableWindow: 300,
				ConfigMaps:   []dto.ConfigMapInfo{},
				Volumes:      []dto.VolumeInfo{},
				Containers:   []dto.TektonContainerInfo{},
			},
			DryRun: "true",
		}

		ctx := context.Background()
		_, err := client.TriggerDeploy(ctx, expectedRequest)

		require.NoError(t, err)

		// Verify the request was sent correctly
		assert.Equal(t, expectedRequest.DeploymentConfigJSON.ProjectID, receivedRequest.DeploymentConfigJSON.ProjectID)
		assert.Equal(t, expectedRequest.DeploymentConfigJSON.ServiceName, receivedRequest.DeploymentConfigJSON.ServiceName)
		assert.Equal(t, expectedRequest.DeploymentConfigJSON.Namespace, receivedRequest.DeploymentConfigJSON.Namespace)
		assert.Equal(t, expectedRequest.DeploymentConfigJSON.StableWindow, receivedRequest.DeploymentConfigJSON.StableWindow)
		assert.Equal(t, expectedRequest.DryRun, receivedRequest.DryRun)
	})
}
