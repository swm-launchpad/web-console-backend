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

	t.Run("전체 볼륨 플로우: 프로젝트 생성 → 볼륨 추가 → 조회 → 수정 → 삭제", func(t *testing.T) {
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
			"name": "Volume Test Project",
			"slug": "volume-test-project",
		}

		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, token)
		assert.Equal(t, http.StatusCreated, w.Code)

		var projectResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &projectResp)
		require.NoError(t, err)

		projectID := uint(projectResp["data"].(map[string]interface{})["project_id"].(float64))

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
		t.Logf("Created volume with ID: %d", volumeID)
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

		// Step 6: 볼륨 수정
		updateVolumeReq := map[string]interface{}{
			"name":     "updated-data-volume",
			"capacity": 200,
		}

		w = server.MakeAuthRequest("PUT", fmt.Sprintf("/api/v1/volumes/%d", volumeID), updateVolumeReq, token)
		if w.Code != http.StatusOK {
			t.Logf("Update volume failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusOK, w.Code)

		var updateVolumeResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &updateVolumeResp)
		require.NoError(t, err)

		updatedVolume := updateVolumeResp["data"].(map[string]interface{})
		assert.Equal(t, "updated-data-volume", updatedVolume["name"])
		assert.Equal(t, float64(200), updatedVolume["capacity"])

		// Step 7: 프로젝트 상세 조회로 볼륨 확인
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/projects/%d", projectID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var projectDetailResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &projectDetailResp)
		require.NoError(t, err)

		projectVolumes := projectDetailResp["data"].(map[string]interface{})["volumes"].([]interface{})
		assert.Equal(t, 2, len(projectVolumes))

		// Step 8: 볼륨 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/volumes/%d", volumeID), nil, token)
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
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/projects/%d", projectID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)
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
			"name": "Constraint Test Project",
			"slug": "constraint-test",
		}
		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, token)
		var projectResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &projectResp)
		projectID := uint(projectResp["data"].(map[string]interface{})["project_id"].(float64))

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

		// Test 3: 다른 프로젝트의 볼륨 수정 시도 (실패해야 함)
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

		// 다른 사용자가 볼륨 수정 시도
		updateReq := map[string]interface{}{
			"name": "hacked-volume",
		}
		w = server.MakeAuthRequest("PUT", "/api/v1/volumes/1", updateReq, otherToken)
		// 권한 체크로 인해 403 Forbidden 또는 404 Not Found 응답을 받아야 함
		assert.True(t, w.Code == http.StatusForbidden || w.Code == http.StatusNotFound,
			"Expected 403 or 404, got %d", w.Code)
	})
}
