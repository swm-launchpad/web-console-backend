package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

	t.Run("전체 볼륨 플로우: 프로젝트 생성 → 볼륨 추가 → 조회 → 삭제 (수정 불가)", func(t *testing.T) {
		// Step 1: 사용자 등록
		registerReq := map[string]string{
			"username": "volumeuser",
			"password": "TestPassword123!",
			"email":    "volume@example.com",
			"name":     "Volume Test User",
		}

		w := server.MakeRequest("POST", "/auth/register", registerReq)
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
		if w.Code != http.StatusCreated {
			t.Logf("Create project failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var projectResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &projectResp)
		require.NoError(t, err)

		t.Logf("Project response: %+v", projectResp)
		projectID := uint(projectResp["data"].(map[string]interface{})["project_id"].(float64))
		projectSlug := projectResp["data"].(map[string]interface{})["slug"].(string)
		t.Logf("Created project with ID: %d, Slug: %s", projectID, projectSlug)

		// Step 3: 볼륨 추가
		addVolumeReq := map[string]interface{}{
			"project_id": projectID,
			"name":       "data-volume",
			"capacity":   256,
		}

		w = server.MakeAuthRequest("POST", "/api/v1/volumes", addVolumeReq, token)
		if w.Code != http.StatusCreated {
			t.Logf("Add volume failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var addVolumeResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &addVolumeResp)
		require.NoError(t, err)

		volumeData := addVolumeResp["data"].(map[string]interface{})
		volumeID := uint(volumeData["volume_id"].(float64))
		volumeSlug := volumeData["slug"].(string)
		t.Logf("Created volume with ID: %d, Slug: %s", volumeID, volumeSlug)
		assert.Equal(t, "data-volume", volumeData["name"])
		assert.Equal(t, float64(256), volumeData["capacity"])

		// Step 4: 두 번째 볼륨 추가
		addVolumeReq2 := map[string]interface{}{
			"project_id": projectID,
			"name":       "backup-volume",
			"capacity":   128,
		}

		w = server.MakeAuthRequest("POST", "/api/v1/volumes", addVolumeReq2, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Step 5: 프로젝트의 볼륨 목록 조회
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/volumes?project_id=%d", projectID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var listVolumesResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &listVolumesResp)
		require.NoError(t, err)

		volumes := listVolumesResp["data"].(map[string]interface{})["volumes"].([]interface{})
		assert.Equal(t, 2, len(volumes))

		// Step 6: 볼륨 수정 시도 (404 반환 - 엔드포인트가 없음)
		updateVolumeReq := map[string]interface{}{
			"name":     "updated-data-volume",
			"capacity": 200,
		}

		w = server.MakeAuthRequest("PUT", fmt.Sprintf("/api/v1/volumes/%s", volumeSlug), updateVolumeReq, token)
		t.Logf("Update volume response with status %d: %s", w.Code, w.Body.String())
		// 볼륨 업데이트 엔드포인트가 없어서 404가 반환됨
		assert.Equal(t, http.StatusNotFound, w.Code, "Volume update endpoint does not exist")

		// Step 7: 프로젝트 상세 조회로 볼륨 확인
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/projects/%s", projectSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var projectDetailResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &projectDetailResp)
		require.NoError(t, err)

		projectVolumes := projectDetailResp["data"].(map[string]interface{})["volumes"].([]interface{})
		assert.Equal(t, 2, len(projectVolumes))

		// Step 8: 볼륨 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/volumes/%s", volumeSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var deleteVolumeResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &deleteVolumeResp)
		require.NoError(t, err)

		assert.True(t, deleteVolumeResp["success"].(bool))
		assert.Equal(t, "Volume removed successfully", deleteVolumeResp["data"].(map[string]interface{})["message"])

		// Step 9: 볼륨 목록 재조회 (1개만 있어야 함)
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/volumes?project_id=%d", projectID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var finalListResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &finalListResp)
		require.NoError(t, err)

		finalVolumes := finalListResp["data"].(map[string]interface{})["volumes"].([]interface{})
		assert.Equal(t, 1, len(finalVolumes))
		assert.Equal(t, "backup-volume", finalVolumes[0].(map[string]interface{})["name"])

		// Step 10: 프로젝트 삭제 (관련 볼륨도 모두 삭제되어야 함)
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", projectSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		// Step 11: 프로젝트가 삭제된 후 볼륨 목록 조회 (볼륨이 모두 삭제되었는지 확인)
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/volumes?project_id=%d", projectID), nil, token)

		// 프로젝트가 삭제된 후에는 해당 프로젝트의 볼륨 조회가 404를 반환하거나 빈 목록을 반환
		if w.Code == http.StatusOK {
			var deletedProjectVolumesResp map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &deletedProjectVolumesResp)
			require.NoError(t, err)

			// 볼륨이 모두 삭제되어 빈 배열이어야 함
			deletedVolumes := deletedProjectVolumesResp["data"].(map[string]interface{})["volumes"].([]interface{})
			assert.Equal(t, 0, len(deletedVolumes), "All volumes should be deleted when project is deleted")
		} else {
			// 404 Not Found도 볼륨이 삭제되었음을 의미
			assert.Equal(t, http.StatusNotFound, w.Code, "Volumes should return 404 when project is deleted")
		}
	})
}

func TestVolumeConstraints_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// 테스트 서버 설정
	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("볼륨 제약 조건 테스트", func(t *testing.T) {
		// 사용자 및 프로젝트 설정
		registerReq := map[string]string{
			"username": "constraintuser",
			"password": "TestPassword123!",
			"email":    "constraint@example.com",
		}
		w := server.MakeRequest("POST", "/auth/register", registerReq)
		var registerResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &registerResp)
		token := registerResp["data"].(map[string]interface{})["token"].(string)

		createProjectReq := map[string]interface{}{
			"name":          "Constraint Test Project",
			"cpu_limit":     1000,
			"memory_limit":  2048,
			"disk_limit":    2048,
			"traffic_limit": 128,
		}
		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, token)
		var projectResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &projectResp)
		projectID := uint(projectResp["data"].(map[string]interface{})["project_id"].(float64))
		_ = projectResp["data"].(map[string]interface{})["slug"].(string)

		// Test 1: 중복된 볼륨 이름으로 추가 시도 (실패해야 함)
		addVolumeReq := map[string]interface{}{
			"project_id": projectID,
			"name":       "duplicate-volume",
			"capacity":   150,
		}

		w = server.MakeAuthRequest("POST", "/api/v1/volumes", addVolumeReq, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		// 같은 이름으로 다시 추가 시도
		w = server.MakeAuthRequest("POST", "/api/v1/volumes", addVolumeReq, token)
		assert.Equal(t, http.StatusConflict, w.Code)

		// Test 2: 잘못된 용량으로 볼륨 추가 시도 (실패해야 함)
		invalidVolumeReq := map[string]interface{}{
			"project_id": projectID,
			"name":       "invalid-volume",
			"capacity":   0, // Invalid capacity
		}

		w = server.MakeAuthRequest("POST", "/api/v1/volumes", invalidVolumeReq, token)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Test 3: 프로젝트 디스크 제한 초과 볼륨 추가 시도 (실패해야 함)
		// 프로젝트의 disk_limit은 2048Mi
		// 이미 150Mi 볼륨이 있으므로, 2048Mi를 추가하면 총 2198Mi로 초과
		exceedVolumeReq := map[string]interface{}{
			"project_id": projectID,
			"name":       "exceed-disk-limit",
			"capacity":   2048, // 프로젝트 disk_limit과 동일한 용량
		}

		w = server.MakeAuthRequest("POST", "/api/v1/volumes", exceedVolumeReq, token)
		if w.Code != http.StatusBadRequest {
			t.Logf("Exceed volume failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var exceedResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &exceedResp)
		assert.False(t, exceedResp["success"].(bool))
		// 에러 메시지에 "disk" 또는 "limit" 또는 "exceed"가 포함되어야 함
		errorMsg := exceedResp["error"].(map[string]interface{})["message"].(string)
		assert.True(t,
			strings.Contains(strings.ToLower(errorMsg), "disk") ||
				strings.Contains(strings.ToLower(errorMsg), "limit") ||
				strings.Contains(strings.ToLower(errorMsg), "exceed"),
			"Error message should mention disk limit, got: %s", errorMsg)

		// Test 4: 다른 프로젝트의 볼륨 삭제 시도 (실패해야 함)
		// 다른 사용자 생성
		otherReq := map[string]string{
			"username": "othervoluser",
			"password": "OtherPass123!",
			"email":    "othervol@example.com",
		}
		w = server.MakeRequest("POST", "/auth/register", otherReq)
		var otherResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &otherResp)
		otherToken := otherResp["data"].(map[string]interface{})["token"].(string)

		// 볼륨 slug를 가져와서 다른 사용자가 볼륨 수정 시도
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/volumes?project_id=%d", projectID), nil, token)
		var volumeListResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &volumeListResp)
		volumes := volumeListResp["data"].(map[string]interface{})["volumes"].([]interface{})
		existingVolumeSlug := volumes[0].(map[string]interface{})["slug"].(string)

		// 다른 사용자가 볼륨 삭제 시도 (권한으로 인해 실패해야 함)
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/volumes/%s", existingVolumeSlug), nil, otherToken)
		// 권한이 없으므로 404 Not Found 반환 (정보 노출 방지)
		assert.Equal(t, http.StatusNotFound, w.Code,
			"Should return 404 for unauthorized volume access")

		// Test 5: project_id 파라미터 누락 시 볼륨 목록 조회 (실패해야 함)
		w = server.MakeAuthRequest("GET", "/api/v1/volumes", nil, token)
		assert.Equal(t, http.StatusBadRequest, w.Code,
			"Should return 400 when project_id parameter is missing")

		var missingParamResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &missingParamResp)
		assert.False(t, missingParamResp["success"].(bool))
	})
}
