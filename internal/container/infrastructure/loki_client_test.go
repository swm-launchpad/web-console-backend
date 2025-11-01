package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
)

func TestBuildLogQLQuery(t *testing.T) {
	cfg := &config.Config{
		Loki: config.LokiConfig{
			URL:      "https://loki.test",
			Username: "test",
			Password: "pass",
			OrgID:    "fake",
		},
	}
	log := logger.NewForTest()
	client := NewLokiClient(cfg, log)

	tests := []struct {
		name            string
		pipelineRunName string
		excludeTasks    []string
		expected        string
	}{
		{
			name:            "no exclusions",
			pipelineRunName: "image-build-push-run-abc123",
			excludeTasks:    []string{},
			expected:        `{namespace="build-pipeline",pod=~"image-build-push-run-abc123-.*"}`,
		},
		{
			name:            "single exclusion",
			pipelineRunName: "image-build-push-run-xyz789",
			excludeTasks:    []string{"ecr-repository-check"},
			expected:        `{namespace="build-pipeline",pod=~"image-build-push-run-xyz789-.*"} !~ "ecr-repository-check"`,
		},
		{
			name:            "multiple exclusions",
			pipelineRunName: "test-run-123",
			excludeTasks:    []string{"task1", "task2", "task3"},
			expected:        `{namespace="build-pipeline",pod=~"test-run-123-.*"} !~ "task1" !~ "task2" !~ "task3"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := client.buildLogQLQuery(tt.pipelineRunName, tt.excludeTasks)
			assert.Equal(t, tt.expected, query)
		})
	}
}

func TestNewLokiClient(t *testing.T) {
	cfg := &config.Config{
		Loki: config.LokiConfig{
			URL:      "https://loki.example.com",
			Username: "user",
			Password: "pass",
			OrgID:    "tenant-1",
		},
	}
	log := logger.NewForTest()

	client := NewLokiClient(cfg, log)

	assert.NotNil(t, client)
	assert.Equal(t, "https://loki.example.com", client.baseURL)
	assert.Equal(t, "user", client.username)
	assert.Equal(t, "pass", client.password)
	assert.Equal(t, "tenant-1", client.orgID)
}
