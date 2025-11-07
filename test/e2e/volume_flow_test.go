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

func TestVolumeFlow_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// 테스트 서버 설정
	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("전체 볼륨 플로우: 프로젝트 생성 → 컨테이너 생성 → 볼륨 추가 → 조회 → 삭제", func(t *testing.T) {
		// Step 1: 사용자 등록
		registerReq := map[string]string{
			"email":    "volume@example.com",
			"password": "TestPassword123!",
			"nickname": "Volume Test User",
		}

		w := server.MakeRequest("POST", "/api/v1/auth/register", registerReq)
		assert.Equal(t, http.StatusCreated, w.Code)

		var registerResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)

		token := registerResp["data"].(map[string]interface{})["token"].(string)

		// Step 2: 프로젝트 생성
		createProjectReq := map[string]interface{}{
			"name":          "Volume Test Project",
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
			"name":         "volume-test-container",
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

		// Step 4: 볼륨 추가
		addVolumeReq := map[string]interface{}{
			"name":       "data-volume",
			"capacity":   256,
			"mount_path": "/app/data",
		}

		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), addVolumeReq, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		var addVolumeResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &addVolumeResp)
		require.NoError(t, err)

		volumeData := addVolumeResp["data"].(map[string]interface{})
		volumeID := uint(volumeData["volume_id"].(float64))
		t.Logf("Created volume with ID: %d", volumeID)
		assert.Equal(t, "data-volume", volumeData["name"])
		assert.Equal(t, float64(256), volumeData["capacity"])
		assert.Equal(t, "/app/data", volumeData["mount_path"])

		// Step 5: 두 번째 볼륨 추가
		addVolumeReq2 := map[string]interface{}{
			"name":       "backup-volume",
			"capacity":   128,
			"mount_path": "/app/backup",
		}

		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), addVolumeReq2, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Step 6: 컨테이너의 볼륨 목록 조회
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), nil, token)
		t.Logf("List volumes response: %s", w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)

		var listVolumesResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &listVolumesResp)
		require.NoError(t, err)

		t.Logf("Parsed response: %+v", listVolumesResp)
		volumes := listVolumesResp["data"].(map[string]interface{})["volumes"].([]interface{})
		t.Logf("Volumes count: %d", len(volumes))
		assert.Equal(t, 2, len(volumes))

		// Step 7: 볼륨 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/containers/%s/volumes/%d", containerSlug, volumeID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		// Step 8: 볼륨 목록 재조회 (1개만 있어야 함)
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var finalListResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &finalListResp)
		require.NoError(t, err)

		finalVolumes := finalListResp["data"].(map[string]interface{})["volumes"].([]interface{})
		assert.Equal(t, 1, len(finalVolumes))
		assert.Equal(t, "backup-volume", finalVolumes[0].(map[string]interface{})["name"])

		// Step 9: 컨테이너 삭제 (관련 볼륨도 모두 삭제되어야 함)
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/containers/%s", containerSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		// Step 10: 프로젝트 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", projectSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestVolumeConstraints_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("볼륨 제약 조건 테스트", func(t *testing.T) {
		// 사용자 등록 및 인증
		registerReq := map[string]string{
			"email":    "volumeconstraint@example.com",
			"password": "TestPassword123!",
			"nickname": "Volume Constraint User",
		}

		w := server.MakeRequest("POST", "/api/v1/auth/register", registerReq)
		assert.Equal(t, http.StatusCreated, w.Code)

		var registerResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)

		token := registerResp["data"].(map[string]interface{})["token"].(string)

		// 프로젝트 생성
		createProjectReq := map[string]interface{}{
			"name":          "Constraint Test Project",
			"cpu_limit":     1000,
			"memory_limit":  2048,
			"disk_limit":    1024, // 테스트용 디스크 제한
			"traffic_limit": 128,
		}

		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, token)
		if w.Code != http.StatusCreated {
			t.Logf("Create project failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var projectResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &projectResp)
		require.NoError(t, err)

		projectSlug := projectResp["data"].(map[string]interface{})["slug"].(string)

		// 컨테이너 생성
		createContainerReq := map[string]interface{}{
			"name":         "constraint-test-container",
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

		// 테스트 1: 정상적인 볼륨 생성
		addVolumeReq := map[string]interface{}{
			"name":       "test-volume",
			"capacity":   256,
			"mount_path": "/app/data",
		}
		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), addVolumeReq, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		// 테스트 2: 중복된 이름으로 볼륨 생성 시도
		duplicateVolumeReq := map[string]interface{}{
			"name":       "test-volume", // 동일한 이름
			"capacity":   128,
			"mount_path": "/app/data2",
		}
		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), duplicateVolumeReq, token)
		assert.Equal(t, http.StatusConflict, w.Code)

		// 테스트 3: 잘못된 용량 (너무 작음)
		tooSmallVolumeReq := map[string]interface{}{
			"name":       "too-small-volume",
			"capacity":   64, // 최소 128Mi 이하
			"mount_path": "/app/small",
		}
		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), tooSmallVolumeReq, token)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// 테스트 4: 프로젝트 디스크 제한 초과
		exceedVolumeReq := map[string]interface{}{
			"name":       "exceed-volume",
			"capacity":   2000, // 프로젝트 disk_limit인 1024Mi를 초과
			"mount_path": "/app/exceed",
		}
		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), exceedVolumeReq, token)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
