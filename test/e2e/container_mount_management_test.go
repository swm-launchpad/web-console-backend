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

func TestManualMountManagement_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// 테스트 서버 설정
	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("수동으로 볼륨 마운트 추가 및 삭제", func(t *testing.T) {
		// Step 1: 사용자 등록
		registerReq := map[string]string{
			"username": "mountuser",
			"password": "TestPassword123!",
			"email":    "mount@example.com",
			"name":     "Mount Test User",
		}

		w := server.MakeRequest("POST", "/auth/register", registerReq)
		assert.Equal(t, http.StatusCreated, w.Code)

		var registerResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)

		token := registerResp["data"].(map[string]interface{})["token"].(string)

		// Step 2: 프로젝트 생성
		createProjectReq := map[string]interface{}{
			"name":          "Mount Test Project",
			"slug":          "mount-test-project",
			"cpu_limit":     4000,
			"memory_limit":  8192,
			"disk_limit":    10240,
			"traffic_limit": 256,
		}

		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		var projectResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &projectResp)
		require.NoError(t, err)

		projectID := uint(projectResp["data"].(map[string]interface{})["project_id"].(float64))
		t.Logf("Created project with ID: %d", projectID)

		// Step 3: 템플릿 목록 조회 (Express.js 템플릿 찾기 - 볼륨 없는 템플릿)
		w = server.MakeAuthRequest("GET", "/api/v1/templates", nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var templatesResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &templatesResp)
		require.NoError(t, err)

		templates := templatesResp["data"].(map[string]interface{})["templates"].([]interface{})
		var expressTemplateID uint
		for _, tmpl := range templates {
			template := tmpl.(map[string]interface{})
			if template["name"] == "Express.js" {
				expressTemplateID = uint(template["template_id"].(float64))
				break
			}
		}
		require.NotEqual(t, uint(0), expressTemplateID, "Express.js template should exist")
		t.Logf("Found Express.js template with ID: %d", expressTemplateID)

		// Step 4: 컨테이너 생성 (볼륨 없이)
		createContainerReq := map[string]interface{}{
			"name":        "express-app",
			"template_id": expressTemplateID,
			"template_config": map[string]interface{}{
				"node_version": "20",
				"app_port":     "3000",
			},
			"cpu_limit":    1000,
			"memory_limit": 2048,
			"git_url":      "https://github.com/example/express-app",
			"git_branch":   "main",
		}

		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/projects/%d/containers", projectID), createContainerReq, token)
		if w.Code != http.StatusCreated {
			t.Logf("Create container failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var containerResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &containerResp)
		require.NoError(t, err)

		containerID := uint(containerResp["data"].(map[string]interface{})["container_id"].(float64))
		t.Logf("Created container with ID: %d", containerID)

		// Step 5: 볼륨 생성
		createVolumeReq := map[string]interface{}{
			"name":       "test-volume",
			"capacity":   1024,
			"project_id": projectID,
		}

		w = server.MakeAuthRequest("POST", "/api/v1/volumes", createVolumeReq, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		var volumeResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &volumeResp)
		require.NoError(t, err)

		volumeID := uint(volumeResp["data"].(map[string]interface{})["volume_id"].(float64))
		t.Logf("Created volume with ID: %d", volumeID)

		// Step 6: 마운트 추가 (POST /api/v1/containers/:id/mounts)
		addMountReq := map[string]interface{}{
			"volume_id":  volumeID,
			"mount_path": "/data",
		}

		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/containers/%d/mounts", containerID), addMountReq, token)
		if w.Code != http.StatusCreated {
			t.Logf("Add mount failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var mountResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &mountResp)
		require.NoError(t, err)

		mountData := mountResp["data"].(map[string]interface{})
		assert.Equal(t, float64(containerID), mountData["container_id"].(float64))
		assert.Equal(t, float64(volumeID), mountData["volume_id"].(float64))
		assert.Equal(t, "/data", mountData["mount_path"].(string))
		t.Logf("Mount added successfully: volume_id=%d, mount_path=%s", volumeID, mountData["mount_path"].(string))

		// Step 7: 컨테이너 상세 조회하여 마운트 확인
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/containers/%d", containerID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var containerDetailResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &containerDetailResp)
		require.NoError(t, err)

		containerDetail := containerDetailResp["data"].(map[string]interface{})
		mounts, hasMounts := containerDetail["mounts"].([]interface{})
		assert.True(t, hasMounts, "Container should have mounts field")
		assert.Equal(t, 1, len(mounts), "Container should have exactly one mount")

		mount := mounts[0].(map[string]interface{})
		assert.Equal(t, float64(volumeID), mount["volume_id"].(float64))
		assert.Equal(t, "/data", mount["mount_path"].(string))
		t.Logf("Mount verified in container detail")

		// Step 8: 마운트 삭제 (DELETE /api/v1/containers/:id/mounts/:volume_id)
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/containers/%d/mounts/%d", containerID, volumeID), nil, token)
		if w.Code != http.StatusOK {
			t.Logf("Delete mount failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusOK, w.Code)

		t.Logf("Mount deleted successfully")

		// Step 9: 컨테이너 상세 조회하여 마운트가 삭제되었는지 확인
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/containers/%d", containerID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		err = json.Unmarshal(w.Body.Bytes(), &containerDetailResp)
		require.NoError(t, err)

		containerDetail = containerDetailResp["data"].(map[string]interface{})
		mounts, hasMounts = containerDetail["mounts"].([]interface{})
		if hasMounts {
			assert.Equal(t, 0, len(mounts), "Container should have no mounts after deletion")
		}
		t.Logf("Mount deletion verified")

		// Step 10: 컨테이너 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/containers/%d", containerID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		// Step 11: 볼륨 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/volumes/%d", volumeID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		// Step 12: 프로젝트 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/projects/%d", projectID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("중복 마운트 경로 시도 시 실패", func(t *testing.T) {
		// Step 1: 사용자 등록
		registerReq := map[string]string{
			"username": "dupemountuser",
			"password": "TestPassword123!",
			"email":    "dupemount@example.com",
			"name":     "Dupe Mount User",
		}

		w := server.MakeRequest("POST", "/auth/register", registerReq)
		assert.Equal(t, http.StatusCreated, w.Code)

		var registerResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)

		token := registerResp["data"].(map[string]interface{})["token"].(string)

		// Step 2: 프로젝트 생성
		createProjectReq := map[string]interface{}{
			"name":          "Dupe Mount Project",
			"slug":          "dupe-mount-project",
			"cpu_limit":     2000,
			"memory_limit":  4096,
			"disk_limit":    5120,
			"traffic_limit": 128,
		}

		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		var projectResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &projectResp)
		require.NoError(t, err)

		projectID := uint(projectResp["data"].(map[string]interface{})["project_id"].(float64))

		// Step 3: 컨테이너 생성
		w = server.MakeAuthRequest("GET", "/api/v1/templates", nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var templatesResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &templatesResp)
		require.NoError(t, err)

		templates := templatesResp["data"].(map[string]interface{})["templates"].([]interface{})
		var expressTemplateID uint
		for _, tmpl := range templates {
			template := tmpl.(map[string]interface{})
			if template["name"] == "Express.js" {
				expressTemplateID = uint(template["template_id"].(float64))
				break
			}
		}

		createContainerReq := map[string]interface{}{
			"name":         "test-app",
			"template_id":  expressTemplateID,
			"cpu_limit":    1000,
			"memory_limit": 2048,
			"git_url":      "https://github.com/example/app",
			"git_branch":   "main",
		}

		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/projects/%d/containers", projectID), createContainerReq, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		var containerResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &containerResp)
		require.NoError(t, err)

		containerID := uint(containerResp["data"].(map[string]interface{})["container_id"].(float64))

		// Step 4: 두 개의 볼륨 생성
		createVolume1Req := map[string]interface{}{
			"name":       "volume-1",
			"capacity":   512,
			"project_id": projectID,
		}

		w = server.MakeAuthRequest("POST", "/api/v1/volumes", createVolume1Req, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		var volume1Resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &volume1Resp)
		require.NoError(t, err)

		volume1ID := uint(volume1Resp["data"].(map[string]interface{})["volume_id"].(float64))

		createVolume2Req := map[string]interface{}{
			"name":       "volume-2",
			"capacity":   512,
			"project_id": projectID,
		}

		w = server.MakeAuthRequest("POST", "/api/v1/volumes", createVolume2Req, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		var volume2Resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &volume2Resp)
		require.NoError(t, err)

		volume2ID := uint(volume2Resp["data"].(map[string]interface{})["volume_id"].(float64))

		// Step 5: 첫 번째 볼륨을 /data에 마운트
		addMount1Req := map[string]interface{}{
			"volume_id":  volume1ID,
			"mount_path": "/data",
		}

		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/containers/%d/mounts", containerID), addMount1Req, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Step 6: 두 번째 볼륨을 동일한 경로(/data)에 마운트 시도 - 실패해야 함
		addMount2Req := map[string]interface{}{
			"volume_id":  volume2ID,
			"mount_path": "/data",
		}

		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/containers/%d/mounts", containerID), addMount2Req, token)
		assert.NotEqual(t, http.StatusCreated, w.Code, "Should fail to mount to duplicate path")
		t.Logf("Duplicate mount path correctly rejected with status %d", w.Code)

		// Cleanup
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/containers/%d", containerID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/volumes/%d", volume1ID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/volumes/%d", volume2ID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/projects/%d", projectID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
