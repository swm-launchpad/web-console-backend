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

func TestVolumeSecurity_AddVolume_UnauthorizedUser_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("권한 없는 사용자는 다른 프로젝트의 컨테이너에 volume 추가 불가", func(t *testing.T) {
		// User1: 프로젝트 소유자
		user1RegisterReq := map[string]string{
			"email":    "owner_security1@example.com",
			"password": "TestPassword123!",
			"nickname": "Owner User 1",
		}
		w := server.MakeRequest("POST", "/api/v1/auth/register", user1RegisterReq)
		require.Equal(t, http.StatusCreated, w.Code)

		var user1Resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &user1Resp)
		require.NoError(t, err)
		user1Token := user1Resp["data"].(map[string]interface{})["token"].(string)

		// User1의 프로젝트 생성
		createProjectReq := map[string]interface{}{
			"name":          "Security Test Project 1",
			"cpu_limit":     1000,
			"memory_limit":  2048,
			"disk_limit":    2048,
			"traffic_limit": 128,
		}
		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, user1Token)
		require.Equal(t, http.StatusCreated, w.Code)

		var projectResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &projectResp)
		require.NoError(t, err)
		projectSlug := projectResp["data"].(map[string]interface{})["slug"].(string)

		// User1의 컨테이너 생성
		createContainerReq := map[string]interface{}{
			"name":         "secure-container",
			"git_url":      "https://github.com/test/repo",
			"git_branch":   "main",
			"cpu_limit":    500,
			"memory_limit": 1024,
			"template_id":  1,
		}
		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/projects/%s/containers", projectSlug), createContainerReq, user1Token)
		require.Equal(t, http.StatusCreated, w.Code)

		var containerResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &containerResp)
		require.NoError(t, err)
		containerSlug := containerResp["data"].(map[string]interface{})["slug"].(string)
		t.Logf("Created container: %s", containerSlug)

		// User2: 공격자 (권한 없음)
		user2RegisterReq := map[string]string{
			"email":    "hacker_security1@example.com",
			"password": "TestPassword123!",
			"nickname": "Hacker User",
		}
		w = server.MakeRequest("POST", "/api/v1/auth/register", user2RegisterReq)
		require.Equal(t, http.StatusCreated, w.Code)

		var user2Resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &user2Resp)
		require.NoError(t, err)
		user2Token := user2Resp["data"].(map[string]interface{})["token"].(string)

		// User2가 User1의 container에 volume 추가 시도 (should FAIL)
		addVolumeReq := map[string]interface{}{
			"name":       "hacker-volume",
			"capacity":   512,
			"mount_path": "/hacked",
		}
		w = server.MakeAuthRequest("POST",
			fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug),
			addVolumeReq, user2Token)

		// ✅ 권한 없음 - 403 또는 404 기대
		assert.NotEqual(t, http.StatusCreated, w.Code,
			"Unauthorized user should NOT be able to add volume")
		t.Logf("Unauthorized add volume returned: %d", w.Code)

		// ✅ Orphan volume이 생성되지 않았는지 확인
		// User1의 project에 volume 조회
		w = server.MakeAuthRequest("GET",
			fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug),
			nil, user1Token)
		require.Equal(t, http.StatusOK, w.Code)

		var volumesResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &volumesResp)
		require.NoError(t, err)

		volumes := volumesResp["data"].(map[string]interface{})["volumes"].([]interface{})
		assert.Equal(t, 0, len(volumes),
			"No orphan volume should be created after unauthorized attempt")
		t.Logf("✅ Security check passed: No orphan volumes created")
	})
}

func TestVolumeSecurity_AddVolume_MountFailureRollback_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("Mount 실패 시 volume이 rollback되어 orphan volume이 생성되지 않음", func(t *testing.T) {
		// 사용자 등록
		registerReq := map[string]string{
			"email":    "rollback@example.com",
			"password": "TestPassword123!",
			"nickname": "Rollback Test User",
		}
		w := server.MakeRequest("POST", "/api/v1/auth/register", registerReq)
		require.Equal(t, http.StatusCreated, w.Code)

		var registerResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)
		token := registerResp["data"].(map[string]interface{})["token"].(string)

		// 프로젝트 생성
		createProjectReq := map[string]interface{}{
			"name":          "Rollback Test Project",
			"cpu_limit":     1000,
			"memory_limit":  2048,
			"disk_limit":    2048,
			"traffic_limit": 128,
		}
		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, token)
		require.Equal(t, http.StatusCreated, w.Code)

		var projectResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &projectResp)
		require.NoError(t, err)
		projectSlug := projectResp["data"].(map[string]interface{})["slug"].(string)

		// 컨테이너 생성
		createContainerReq := map[string]interface{}{
			"name":         "rollback-container",
			"git_url":      "https://github.com/test/repo",
			"git_branch":   "main",
			"cpu_limit":    500,
			"memory_limit": 1024,
			"template_id":  1,
		}
		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/projects/%s/containers", projectSlug), createContainerReq, token)
		require.Equal(t, http.StatusCreated, w.Code)

		var containerResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &containerResp)
		require.NoError(t, err)
		containerSlug := containerResp["data"].(map[string]interface{})["slug"].(string)

		// 첫 번째 volume 추가 (성공)
		addVolume1Req := map[string]interface{}{
			"name":       "volume1",
			"capacity":   512,
			"mount_path": "/data",
		}
		w = server.MakeAuthRequest("POST",
			fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug),
			addVolume1Req, token)
		require.Equal(t, http.StatusCreated, w.Code)
		t.Logf("✅ First volume added successfully")

		// 두 번째 volume - 같은 mount path (실패 예상)
		addVolume2Req := map[string]interface{}{
			"name":       "volume2",
			"capacity":   512,
			"mount_path": "/data", // ❌ 중복 mount path
		}
		w = server.MakeAuthRequest("POST",
			fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug),
			addVolume2Req, token)

		// ✅ Mount 실패 확인
		assert.NotEqual(t, http.StatusCreated, w.Code,
			"Adding volume with duplicate mount path should fail")
		t.Logf("Duplicate mount path returned: %d", w.Code)

		// ✅ volume2가 orphan으로 남지 않았는지 확인
		w = server.MakeAuthRequest("GET",
			fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug),
			nil, token)
		require.Equal(t, http.StatusOK, w.Code)

		var volumesResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &volumesResp)
		require.NoError(t, err)

		volumes := volumesResp["data"].(map[string]interface{})["volumes"].([]interface{})
		assert.Equal(t, 1, len(volumes),
			"Only first volume should exist, second should be rolled back")

		if len(volumes) > 0 {
			firstVolume := volumes[0].(map[string]interface{})
			assert.Equal(t, "volume1", firstVolume["name"],
				"Only the first volume should remain")
		}
		t.Logf("✅ Rollback check passed: Only 1 volume exists (volume2 was rolled back)")
	})
}

func TestVolumeSecurity_ListVolumes_UnauthorizedAccess_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("권한 없는 사용자는 다른 프로젝트의 volume 정보를 조회할 수 없음", func(t *testing.T) {
		// User1: 프로젝트 소유자
		user1RegisterReq := map[string]string{
			"email":    "owner_security2@example.com",
			"password": "TestPassword123!",
			"nickname": "Owner User 2",
		}
		w := server.MakeRequest("POST", "/api/v1/auth/register", user1RegisterReq)
		require.Equal(t, http.StatusCreated, w.Code)

		var user1Resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &user1Resp)
		require.NoError(t, err)
		user1Token := user1Resp["data"].(map[string]interface{})["token"].(string)

		// 프로젝트 생성
		createProjectReq := map[string]interface{}{
			"name":          "Private Project",
			"cpu_limit":     1000,
			"memory_limit":  2048,
			"disk_limit":    2048,
			"traffic_limit": 128,
		}
		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, user1Token)
		require.Equal(t, http.StatusCreated, w.Code)

		var projectResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &projectResp)
		require.NoError(t, err)
		projectSlug := projectResp["data"].(map[string]interface{})["slug"].(string)

		// 컨테이너 생성
		createContainerReq := map[string]interface{}{
			"name":         "private-container",
			"git_url":      "https://github.com/test/repo",
			"git_branch":   "main",
			"cpu_limit":    500,
			"memory_limit": 1024,
			"template_id":  1,
		}
		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/projects/%s/containers", projectSlug), createContainerReq, user1Token)
		require.Equal(t, http.StatusCreated, w.Code)

		var containerResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &containerResp)
		require.NoError(t, err)
		containerSlug := containerResp["data"].(map[string]interface{})["slug"].(string)

		// Volume 추가 (secret 정보)
		addVolumeReq := map[string]interface{}{
			"name":       "secret-volume",
			"capacity":   512,
			"mount_path": "/secrets",
		}
		w = server.MakeAuthRequest("POST",
			fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug),
			addVolumeReq, user1Token)
		require.Equal(t, http.StatusCreated, w.Code)
		t.Logf("✅ Secret volume added by owner")

		// User2: 공격자 (권한 없음)
		user2RegisterReq := map[string]string{
			"email":    "spy_security2@example.com",
			"password": "TestPassword123!",
			"nickname": "Spy User",
		}
		w = server.MakeRequest("POST", "/api/v1/auth/register", user2RegisterReq)
		require.Equal(t, http.StatusCreated, w.Code)

		var user2Resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &user2Resp)
		require.NoError(t, err)
		user2Token := user2Resp["data"].(map[string]interface{})["token"].(string)

		// User2가 User1의 volume 조회 시도 (should FAIL)
		w = server.MakeAuthRequest("GET",
			fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug),
			nil, user2Token)

		// ✅ 권한 없음 - 403 또는 404 기대
		assert.NotEqual(t, http.StatusOK, w.Code,
			"Unauthorized user should NOT be able to list volumes")
		t.Logf("Unauthorized list volumes returned: %d", w.Code)
		t.Logf("✅ Security check passed: Unauthorized access blocked")
	})
}
