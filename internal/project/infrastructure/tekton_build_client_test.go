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

func TestNewTektonBuildClient(t *testing.T) {
	tests := []struct {
		name           string
		buildURL       string
		authHeader     string
		registryURL    string
		expectedError  error
		shouldSetupEnv bool
	}{
		{
			name:           "성공: 유효한 환경변수로 클라이언트 생성",
			buildURL:       "https://test-tekton-api.example.com/build",
			authHeader:     "Basic dGVzdDp0ZXN0",
			registryURL:    "test-registry.example.com",
			expectedError:  nil,
			shouldSetupEnv: true,
		},
		{
			name:           "실패: TEKTON_BUILD_URL 누락",
			buildURL:       "",
			authHeader:     "Basic dGVzdDp0ZXN0",
			registryURL:    "test-registry.example.com",
			expectedError:  projecterrors.ErrTektonUnavailable,
			shouldSetupEnv: true,
		},
		{
			name:           "실패: TEKTON_API_AUTH 누락",
			buildURL:       "https://test-tekton-api.example.com/build",
			authHeader:     "",
			registryURL:    "test-registry.example.com",
			expectedError:  projecterrors.ErrTektonUnavailable,
			shouldSetupEnv: true,
		},
		{
			name:           "실패: REGISTRY_URL 누락",
			buildURL:       "https://test-tekton-api.example.com/build",
			authHeader:     "Basic dGVzdDp0ZXN0",
			registryURL:    "",
			expectedError:  projecterrors.ErrTektonUnavailable,
			shouldSetupEnv: true,
		},
		{
			name:           "실패: 모든 환경변수 누락",
			buildURL:       "",
			authHeader:     "",
			registryURL:    "",
			expectedError:  projecterrors.ErrTektonUnavailable,
			shouldSetupEnv: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSetupEnv {
				// Backup original env vars
				originalBuildURL := os.Getenv("TEKTON_BUILD_URL")
				originalAuthHeader := os.Getenv("TEKTON_API_AUTH")
				originalRegistryURL := os.Getenv("REGISTRY_URL")

				// Set test env vars
				if tt.buildURL != "" {
					require.NoError(t, os.Setenv("TEKTON_BUILD_URL", tt.buildURL))
				} else {
					require.NoError(t, os.Unsetenv("TEKTON_BUILD_URL"))
				}

				if tt.authHeader != "" {
					require.NoError(t, os.Setenv("TEKTON_API_AUTH", tt.authHeader))
				} else {
					require.NoError(t, os.Unsetenv("TEKTON_API_AUTH"))
				}

				if tt.registryURL != "" {
					require.NoError(t, os.Setenv("REGISTRY_URL", tt.registryURL))
				} else {
					require.NoError(t, os.Unsetenv("REGISTRY_URL"))
				}

				// Restore env vars after test
				defer func() {
					if originalBuildURL != "" {
						_ = os.Setenv("TEKTON_BUILD_URL", originalBuildURL)
					} else {
						_ = os.Unsetenv("TEKTON_BUILD_URL")
					}

					if originalAuthHeader != "" {
						_ = os.Setenv("TEKTON_API_AUTH", originalAuthHeader)
					} else {
						_ = os.Unsetenv("TEKTON_API_AUTH")
					}

					if originalRegistryURL != "" {
						_ = os.Setenv("REGISTRY_URL", originalRegistryURL)
					} else {
						_ = os.Unsetenv("REGISTRY_URL")
					}
				}()
			}

			client, err := NewTektonBuildClient(logger.NewForTest())

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

func TestTektonBuildClient_TriggerBuild(t *testing.T) {
	tests := []struct {
		name           string
		request        *dto.TektonBuildRequest
		mockStatusCode int
		mockResponse   interface{}
		expectedError  error
		checkResponse  func(t *testing.T, response *dto.TektonBuildResponse)
	}{
		{
			name: "성공: 정상 빌드 요청 (202 Accepted)",
			request: &dto.TektonBuildRequest{
				ProjectID:            "1",
				ContainerID:          "10",
				ImageName:            "test-app",
				GitHubURL:            "https://github.com/test-org/test-repo",
				GitHubBranch:         "main",
				DirectoryPath:        ".",
				ForceBuild:           "false",
				LastBuildCommitHash:  "abc123def456",
				Template:             "FROM node:18\nCOPY . /app",
				DockerfileConfigJSON: `{"base_image":"node:18"}`,
				BuildEnvJSON:         `{"NODE_ENV":"production"}`,
				RegistryURL:          "registry.launchpad.kr/",
				InstallationID:       "12345678",
			},
			mockStatusCode: http.StatusAccepted,
			mockResponse: dto.TektonBuildResponse{
				EventListener:    "image-build-push",
				EventListenerUID: "abc-123-def-456",
				EventID:          "build-event-123",
				Namespace:        "build-pipeline",
			},
			expectedError: nil,
			checkResponse: func(t *testing.T, response *dto.TektonBuildResponse) {
				assert.Equal(t, "image-build-push", response.EventListener)
				assert.Equal(t, "abc-123-def-456", response.EventListenerUID)
				assert.Equal(t, "build-event-123", response.EventID)
				assert.Equal(t, "build-pipeline", response.Namespace)
			},
		},
		{
			name: "성공: 강제 빌드 모드",
			request: &dto.TektonBuildRequest{
				ProjectID:    "2",
				ContainerID:  "20",
				ImageName:    "test-app",
				GitHubURL:    "https://github.com/test-org/test-repo",
				GitHubBranch: "main",
				ForceBuild:   "true",
				Template:     "FROM python:3.11\nCOPY . /app",
			},
			mockStatusCode: http.StatusAccepted,
			mockResponse: dto.TektonBuildResponse{
				EventListener: "image-build-push",
				EventID:       "build-event-456",
			},
			expectedError: nil,
			checkResponse: func(t *testing.T, response *dto.TektonBuildResponse) {
				assert.Equal(t, "image-build-push", response.EventListener)
				assert.Equal(t, "build-event-456", response.EventID)
			},
		},
		{
			name: "성공: GitHub 레포지토리 없이 템플릿만 사용",
			request: &dto.TektonBuildRequest{
				ProjectID:   "3",
				ContainerID: "30",
				ImageName:   "template-only-app",
				ForceBuild:  "true",
				Template:    "FROM alpine:latest\nRUN echo 'Hello World'",
			},
			mockStatusCode: http.StatusAccepted,
			mockResponse: dto.TektonBuildResponse{
				EventListener: "image-build-push",
				EventID:       "build-event-789",
			},
			expectedError: nil,
			checkResponse: func(t *testing.T, response *dto.TektonBuildResponse) {
				assert.Equal(t, "image-build-push", response.EventListener)
				assert.Equal(t, "build-event-789", response.EventID)
			},
		},
		{
			name: "실패: 잘못된 요청 (400 Bad Request)",
			request: &dto.TektonBuildRequest{
				ProjectID:   "4",
				ContainerID: "40",
				ImageName:   "",
				ForceBuild:  "true",
			},
			mockStatusCode: http.StatusBadRequest,
			mockResponse:   "validation failed: image_name is required",
			expectedError:  projecterrors.ErrTektonBuildFailed,
			checkResponse:  nil,
		},
		{
			name: "실패: 인증 실패 (401 Unauthorized)",
			request: &dto.TektonBuildRequest{
				ProjectID:   "5",
				ContainerID: "50",
				ImageName:   "test-app",
				ForceBuild:  "true",
				Template:    "FROM node:18",
			},
			mockStatusCode: http.StatusUnauthorized,
			mockResponse:   "unauthorized",
			expectedError:  projecterrors.ErrTektonAuthenticationFailed,
			checkResponse:  nil,
		},
		{
			name: "실패: 권한 거부 (403 Forbidden)",
			request: &dto.TektonBuildRequest{
				ProjectID:   "6",
				ContainerID: "60",
				ImageName:   "test-app",
				ForceBuild:  "true",
				Template:    "FROM node:18",
			},
			mockStatusCode: http.StatusForbidden,
			mockResponse:   "forbidden",
			expectedError:  projecterrors.ErrTektonAuthenticationFailed,
			checkResponse:  nil,
		},
		{
			name: "실패: 서버 에러 (500 Internal Server Error)",
			request: &dto.TektonBuildRequest{
				ProjectID:   "7",
				ContainerID: "70",
				ImageName:   "test-app",
				ForceBuild:  "true",
				Template:    "FROM node:18",
			},
			mockStatusCode: http.StatusInternalServerError,
			mockResponse:   "internal server error",
			expectedError:  projecterrors.ErrTektonUnavailable,
			checkResponse:  nil,
		},
		{
			name: "실패: 잘못된 응답 형식",
			request: &dto.TektonBuildRequest{
				ProjectID:   "8",
				ContainerID: "80",
				ImageName:   "test-app",
				ForceBuild:  "true",
				Template:    "FROM node:18",
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
				case dto.TektonBuildResponse:
					_ = json.NewEncoder(w).Encode(v)
				default:
					_ = json.NewEncoder(w).Encode(v)
				}
			}))
			defer server.Close()

			// Create client with mock server URL
			log := logger.NewForTest()
			client := &tektonBuildClient{
				buildURL:    server.URL,
				authHeader:  "Basic test-auth",
				registryURL: "test-registry.example.com",
				httpClient:  server.Client(),
				logger:      log,
			}

			// Execute test
			ctx := context.Background()
			response, err := client.TriggerBuild(ctx, tt.request)

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

func TestTektonBuildClient_TriggerBuild_NetworkError(t *testing.T) {
	t.Run("실패: 네트워크 에러 (서버 연결 불가)", func(t *testing.T) {
		// Create client with invalid URL (no server listening)
		log := logger.NewForTest()
		client := &tektonBuildClient{
			buildURL:    "http://localhost:9999",
			authHeader:  "Basic test-auth",
			registryURL: "test-registry.example.com",
			httpClient:  &http.Client{},
			logger:      log,
		}

		request := &dto.TektonBuildRequest{
			ProjectID:   "9",
			ContainerID: "90",
			ImageName:   "test-app",
			ForceBuild:  "true",
			Template:    "FROM node:18",
		}

		ctx := context.Background()
		response, err := client.TriggerBuild(ctx, request)

		assert.Error(t, err)
		assert.ErrorIs(t, err, projecterrors.ErrTektonUnavailable)
		assert.Nil(t, response)
	})
}

func TestTektonBuildClient_TriggerBuild_ContextCancellation(t *testing.T) {
	t.Run("실패: Context 취소", func(t *testing.T) {
		// Create mock server that delays response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// This should not be reached because context is cancelled immediately
			w.WriteHeader(http.StatusAccepted)
		}))
		defer server.Close()

		log := logger.NewForTest()
		client := &tektonBuildClient{
			buildURL:    server.URL,
			authHeader:  "Basic test-auth",
			registryURL: "test-registry.example.com",
			httpClient:  server.Client(),
			logger:      log,
		}

		request := &dto.TektonBuildRequest{
			ProjectID:   "10",
			ContainerID: "100",
			ImageName:   "test-app",
			ForceBuild:  "true",
			Template:    "FROM node:18",
		}

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		response, err := client.TriggerBuild(ctx, request)

		assert.Error(t, err)
		assert.ErrorIs(t, err, projecterrors.ErrTektonUnavailable)
		assert.Nil(t, response)
	})
}

func TestTektonBuildClient_TriggerBuild_RequestValidation(t *testing.T) {
	t.Run("요청 본문 검증", func(t *testing.T) {
		var receivedRequest dto.TektonBuildRequest

		// Create mock server to capture request
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Decode and capture the request
			err := json.NewDecoder(r.Body).Decode(&receivedRequest)
			require.NoError(t, err)

			// Send success response
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(dto.TektonBuildResponse{
				EventListener: "image-build-push",
				EventID:       "test-event",
			})
		}))
		defer server.Close()

		log := logger.NewForTest()
		client := &tektonBuildClient{
			buildURL:    server.URL,
			authHeader:  "Basic test-auth",
			registryURL: "test-registry.example.com",
			httpClient:  server.Client(),
			logger:      log,
		}

		expectedRequest := &dto.TektonBuildRequest{
			ProjectID:            "11",
			ContainerID:          "110",
			ImageName:            "my-awesome-app",
			GitHubURL:            "https://github.com/myorg/myrepo",
			GitHubBranch:         "develop",
			DirectoryPath:        "./backend",
			ForceBuild:           "false",
			LastBuildCommitHash:  "xyz789abc123",
			Template:             "FROM golang:1.21\nCOPY . /app",
			DockerfileConfigJSON: `{"version":"1.0"}`,
			BuildEnvJSON:         `{"ENV":"staging"}`,
			RegistryURL:          "custom-registry.example.com/",
			InstallationID:       "99887766",
		}

		ctx := context.Background()
		_, err := client.TriggerBuild(ctx, expectedRequest)

		require.NoError(t, err)

		// Verify the request was sent correctly
		assert.Equal(t, expectedRequest.ProjectID, receivedRequest.ProjectID)
		assert.Equal(t, expectedRequest.ContainerID, receivedRequest.ContainerID)
		assert.Equal(t, expectedRequest.ImageName, receivedRequest.ImageName)
		assert.Equal(t, expectedRequest.GitHubURL, receivedRequest.GitHubURL)
		assert.Equal(t, expectedRequest.GitHubBranch, receivedRequest.GitHubBranch)
		assert.Equal(t, expectedRequest.DirectoryPath, receivedRequest.DirectoryPath)
		assert.Equal(t, expectedRequest.ForceBuild, receivedRequest.ForceBuild)
		assert.Equal(t, expectedRequest.LastBuildCommitHash, receivedRequest.LastBuildCommitHash)
		assert.Equal(t, expectedRequest.Template, receivedRequest.Template)
		assert.Equal(t, expectedRequest.DockerfileConfigJSON, receivedRequest.DockerfileConfigJSON)
		assert.Equal(t, expectedRequest.BuildEnvJSON, receivedRequest.BuildEnvJSON)
		assert.Equal(t, expectedRequest.RegistryURL, receivedRequest.RegistryURL)
		assert.Equal(t, expectedRequest.InstallationID, receivedRequest.InstallationID)
	})
}
