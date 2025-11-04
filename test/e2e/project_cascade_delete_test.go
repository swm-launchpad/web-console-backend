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

func TestProjectWithVolumesCascadeDelete_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// 테스트 서버 설정
	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("프로젝트 삭제 시 컨테이너와 볼륨 cascade 삭제 확인", func(t *testing.T) {
		// Step 1: 사용자 등록
		registerReq := map[string]string{
			"username": "cascadeuser",
			"password": "TestPassword123!",
			"email":    "cascade@example.com",
			"name":     "Cascade Test User",
		}

		w := server.MakeRequest("POST", "/auth/register", registerReq)
		assert.Equal(t, http.StatusCreated, w.Code)

		var registerResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)

		token := registerResp["data"].(map[string]interface{})["token"].(string)

		// Step 2: 프로젝트 생성
		createProjectReq := map[string]interface{}{
			"name":          "Cascade Test Project",
			"cpu_limit":     1000,
			"memory_limit":  2048,
			"disk_limit":    2048,
			"traffic_limit": 128,
		}

		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		var projectResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &projectResp)
		require.NoError(t, err)

		projectSlug := projectResp["data"].(map[string]interface{})["slug"].(string)
		t.Logf("Created project with Slug: %s", projectSlug)

		// Step 3: 컨테이너 생성
		createContainerReq := map[string]interface{}{
			"name":         "test-container",
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
		t.Logf("Created container with Slug: %s", containerSlug)

		// Step 4: 컨테이너에 볼륨 3개 추가 (새 API 사용)
		volumeNames := []string{"volume1", "volume2", "volume3"}

		for i, volumeName := range volumeNames {
			addVolumeReq := map[string]interface{}{
				"name":       volumeName,
				"capacity":   128,
				"mount_path": fmt.Sprintf("/data%d", i+1),
			}

			w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), addVolumeReq, token)
			if w.Code != http.StatusCreated {
				t.Logf("Volume creation failed with status %d: %s", w.Code, w.Body.String())
			}
			require.Equal(t, http.StatusCreated, w.Code)

			var volumeResp map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &volumeResp)
			require.NoError(t, err)

			volumeData := volumeResp["data"].(map[string]interface{})
			volumeID := uint(volumeData["volume_id"].(float64))
			assert.Equal(t, volumeName, volumeData["name"])
			t.Logf("Created volume %s with ID: %d", volumeName, volumeID)
		}

		// Step 5: 볼륨이 생성되었는지 확인 (컨테이너의 볼륨 목록 조회)
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var volumesResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &volumesResp)
		require.NoError(t, err)

		volumes := volumesResp["data"].(map[string]interface{})["volumes"].([]interface{})
		assert.Equal(t, 3, len(volumes), "Should have 3 volumes before deletion")
		t.Logf("Verified container has 3 volumes")

		// Step 6: 프로젝트 삭제 (컨테이너와 볼륨이 cascade 삭제되어야 함)
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", projectSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var deleteResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &deleteResp)
		require.NoError(t, err)

		assert.True(t, deleteResp["success"].(bool))
		assert.Equal(t, "Project deleted successfully", deleteResp["data"].(map[string]interface{})["message"])
		t.Logf("Project deleted successfully")

		// Step 7: Cascade 삭제 검증
		// 7-1: 프로젝트가 삭제되었는지 확인
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/projects/%s", projectSlug), nil, token)
		assert.Equal(t, http.StatusNotFound, w.Code)
		t.Logf("Verified project is deleted (404)")

		// 7-2: 컨테이너가 삭제되었는지 확인
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/containers/%s", containerSlug), nil, token)
		// 컨테이너는 404 또는 400을 반환할 수 있음 (slug 형식 검증에 따라)
		assert.NotEqual(t, http.StatusOK, w.Code, "Container should not be accessible after project deletion")
		t.Logf("Verified container is deleted (status: %d)", w.Code)

		// 7-3: 볼륨이 삭제되었는지 확인 (컨테이너 삭제 시 cascade 삭제되므로 조회 불가 또는 빈 배열)
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), nil, token)
		// 컨테이너가 삭제되었으므로 404 반환하거나, 200 OK로 빈 배열 반환 가능
		if w.Code == http.StatusOK {
			var volumesCheckResp map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &volumesCheckResp)
			require.NoError(t, err)
			volumes := volumesCheckResp["data"].(map[string]interface{})["volumes"].([]interface{})
			assert.Equal(t, 0, len(volumes), "All volumes should be cascade deleted")
			t.Logf("Verified volumes are cascade deleted (empty list)")
		} else {
			assert.NotEqual(t, http.StatusOK, w.Code, "Volumes should not be accessible after container deletion")
			t.Logf("Verified volumes are cascade deleted (status: %d)", w.Code)
		}

		t.Logf("✅ Cascade delete test completed successfully")
		t.Logf("   - Project deleted")
		t.Logf("   - Container cascade deleted")
		t.Logf("   - All 3 volumes cascade deleted")
	})
}
