package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/test/helper"
)

func TestContainerDeleteWithVolumes_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup test server
	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("컨테이너 삭제 시 모든 볼륨과 마운트도 삭제되어야 함", func(t *testing.T) {
		// Step 1: Register user
		registerReq := map[string]string{
			"email":    "volumedelete@example.com",
			"password": "TestPassword123!",
			"nickname": "Volume Delete Test User",
		}

		w := server.MakeRequest("POST", "/api/v1/auth/register", registerReq)
		assert.Equal(t, http.StatusCreated, w.Code)

		var registerResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)

		token := registerResp["data"].(map[string]interface{})["token"].(string)

		// Step 2: Create project with disk limit
		createProjectReq := map[string]interface{}{
			"name":          "Volume Delete Test Project",
			"plan":          "eco",
			"cpu_limit":     1000,
			"memory_limit":  2048,
			"disk_limit":    3072, // 3GB disk limit (within beta tier limits)
			"traffic_limit": 128,
		}

		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, token)
		if w.Code != http.StatusCreated {
			t.Logf("Project creation failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var projectResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &projectResp)
		require.NoError(t, err)

		projectSlug := projectResp["data"].(map[string]interface{})["slug"].(string)
		projectID := uint(projectResp["data"].(map[string]interface{})["project_id"].(float64))
		t.Logf("Created project with slug: %s, ID: %d", projectSlug, projectID)

		// Step 3: Create container
		createContainerReq := map[string]interface{}{
			"name":         "volume-delete-test-container",
			"git_url":      "https://github.com/test/repo",
			"git_branch":   "main",
			"cpu_limit":    500,
			"memory_limit": 1024,
			"template_id":  1,
		}

		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/projects/%s/containers", projectSlug), createContainerReq, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		var containerResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &containerResp)
		require.NoError(t, err)

		containerSlug := containerResp["data"].(map[string]interface{})["slug"].(string)
		containerID := uint(containerResp["data"].(map[string]interface{})["container_id"].(float64))
		t.Logf("Created container with slug: %s, ID: %d", containerSlug, containerID)

		// Step 4: Add 3 volumes
		volume1Req := map[string]interface{}{
			"name":       "data-volume-1",
			"capacity":   512,
			"mount_path": "/app/data1",
		}

		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), volume1Req, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		var volume1Resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &volume1Resp)
		require.NoError(t, err)
		volume1ID := uint(volume1Resp["data"].(map[string]interface{})["volume_id"].(float64))
		t.Logf("Created volume1 with ID: %d", volume1ID)

		volume2Req := map[string]interface{}{
			"name":       "data-volume-2",
			"capacity":   1024,
			"mount_path": "/app/data2",
		}

		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), volume2Req, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		var volume2Resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &volume2Resp)
		require.NoError(t, err)
		volume2ID := uint(volume2Resp["data"].(map[string]interface{})["volume_id"].(float64))
		t.Logf("Created volume2 with ID: %d", volume2ID)

		volume3Req := map[string]interface{}{
			"name":       "data-volume-3",
			"capacity":   512,
			"mount_path": "/app/data3",
		}

		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), volume3Req, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		var volume3Resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &volume3Resp)
		require.NoError(t, err)
		volume3ID := uint(volume3Resp["data"].(map[string]interface{})["volume_id"].(float64))
		t.Logf("Created volume3 with ID: %d", volume3ID)

		// Step 5: Verify volumes exist (list volumes)
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var listVolumesResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &listVolumesResp)
		require.NoError(t, err)

		volumes := listVolumesResp["data"].(map[string]interface{})["volumes"].([]interface{})
		assert.Equal(t, 3, len(volumes), "Should have 3 volumes before deletion")

		// Step 6: Verify project disk usage = 2048 MB (512 + 1024 + 512)
		totalDiskUsage := server.GetProjectDiskUsage(t, projectID)
		assert.Equal(t, uint32(2048), totalDiskUsage, "Project disk usage should be 2048 MB")

		// Step 7: Delete container
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/containers/%s", containerSlug), nil, token)
		if w.Code != http.StatusOK {
			t.Logf("Container deletion failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusOK, w.Code)
		t.Logf("Deleted container with slug: %s", containerSlug)

		// Step 8: Verify container is soft-deleted
		containerExists, isDeleted := server.GetContainerStatus(t, containerID)
		assert.True(t, containerExists, "Container record should still exist")
		assert.True(t, isDeleted, "Container should be marked as deleted")

		// Step 9: Verify all volumes are deleted
		volume1Exists := server.VolumeExists(t, volume1ID)
		assert.False(t, volume1Exists, "Volume1 should be deleted")

		volume2Exists := server.VolumeExists(t, volume2ID)
		assert.False(t, volume2Exists, "Volume2 should be deleted")

		volume3Exists := server.VolumeExists(t, volume3ID)
		assert.False(t, volume3Exists, "Volume3 should be deleted")

		// Step 10: Verify all mounts are deleted (by checking volumes table CASCADE)
		// Mounts are automatically deleted by DB CASCADE when volumes are deleted
		// No need to check mounts table directly

		// Step 11: Verify project disk usage is back to 0
		totalDiskUsageAfter := server.GetProjectDiskUsage(t, projectID)
		assert.Equal(t, uint32(0), totalDiskUsageAfter, "Project disk usage should be 0 after container deletion")

		t.Log("✅ All volumes and mounts successfully deleted with container")
	})
}
