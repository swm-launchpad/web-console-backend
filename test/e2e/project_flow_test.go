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

func TestProjectFlow_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// 테스트 서버 설정
	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("전체 프로젝트 플로우: 생성 → 조회 → 수정 → 삭제", func(t *testing.T) {
		// Step 1: 사용자 등록 (프로젝트 생성을 위한 사용자)
		registerReq := map[string]string{
			"username": "projectuser",
			"password": "TestPassword123!",
			"email":    "project@example.com",
			"name":     "Project Test User",
		}

		w := server.MakeRequest("POST", "/auth/register", registerReq)
		assert.Equal(t, http.StatusCreated, w.Code)

		var registerResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)

		token := registerResp["data"].(map[string]interface{})["token"].(string)

		// Step 2: 프로젝트 생성
		createProjectReq := map[string]interface{}{
			"name": "Test Project",
			"slug": "test-project",
			"fqdn": "test.example.com",
			"plan": "starter",
		}

		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, token)
		if w.Code != http.StatusCreated {
			t.Logf("Create project failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var createResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &createResp)
		require.NoError(t, err)

		assert.True(t, createResp["success"].(bool))
		projectData := createResp["data"].(map[string]interface{})
		projectID := uint(projectData["project_id"].(float64))
		assert.Equal(t, "Test Project", projectData["name"])
		assert.Equal(t, "test-project", projectData["slug"])

		// Step 3: 프로젝트 목록 조회
		w = server.MakeAuthRequest("GET", "/api/v1/projects", nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var listResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &listResp)
		require.NoError(t, err)

		projects := listResp["data"].(map[string]interface{})["projects"].([]interface{})
		assert.Greater(t, len(projects), 0)

		// Step 4: 프로젝트 상세 조회 (by ID)
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/projects/%d", projectID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var getResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &getResp)
		require.NoError(t, err)

		projectDetail := getResp["data"].(map[string]interface{})
		assert.Equal(t, float64(projectID), projectDetail["project_id"])
		assert.Equal(t, "Test Project", projectDetail["name"])

		// Step 5: 프로젝트 상세 조회 (by slug)
		w = server.MakeAuthRequest("GET", "/api/v1/projects/test-project", nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		// Step 6: 프로젝트 수정
		updateProjectReq := map[string]interface{}{
			"name": "Updated Project",
			"fqdn": "updated.example.com",
		}

		w = server.MakeAuthRequest("PUT", fmt.Sprintf("/api/v1/projects/%d", projectID), updateProjectReq, token)
		if w.Code != http.StatusOK {
			t.Logf("Update project failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusOK, w.Code)

		var updateResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &updateResp)
		require.NoError(t, err)

		updatedData := updateResp["data"].(map[string]interface{})
		assert.Equal(t, "Updated Project", updatedData["name"])
		assert.Equal(t, "updated.example.com", updatedData["fqdn"])

		// Step 7: 프로젝트 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/projects/%d", projectID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var deleteResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &deleteResp)
		require.NoError(t, err)

		assert.True(t, deleteResp["success"].(bool))
		assert.Equal(t, "Project deleted successfully", deleteResp["data"].(map[string]interface{})["message"])

		// Step 8: 삭제된 프로젝트 조회 시도 (실패해야 함)
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/projects/%d", projectID), nil, token)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestProjectPermissions_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// 테스트 서버 설정
	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("프로젝트 권한 테스트", func(t *testing.T) {
		// Owner 사용자 생성
		ownerReq := map[string]string{
			"username": "owner",
			"password": "OwnerPass123!",
			"email":    "owner@example.com",
		}
		w := server.MakeRequest("POST", "/auth/register", ownerReq)
		assert.Equal(t, http.StatusCreated, w.Code)
		var ownerResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &ownerResp)
		ownerToken := ownerResp["data"].(map[string]interface{})["token"].(string)

		// Other 사용자 생성
		otherReq := map[string]string{
			"username": "other",
			"password": "OtherPass123!",
			"email":    "other@example.com",
		}
		w = server.MakeRequest("POST", "/auth/register", otherReq)
		assert.Equal(t, http.StatusCreated, w.Code)
		var otherResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &otherResp)
		otherToken := otherResp["data"].(map[string]interface{})["token"].(string)

		// Owner가 프로젝트 생성
		createReq := map[string]interface{}{
			"name": "Owner Project",
			"slug": "owner-project",
		}
		w = server.MakeAuthRequest("POST", "/api/v1/projects", createReq, ownerToken)
		assert.Equal(t, http.StatusCreated, w.Code)
		var createResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &createResp)
		projectID := uint(createResp["data"].(map[string]interface{})["project_id"].(float64))

		// Other 사용자가 프로젝트 수정 시도 (실패해야 함)
		updateReq := map[string]interface{}{
			"name": "Hacked Project",
		}
		w = server.MakeAuthRequest("PUT", fmt.Sprintf("/api/v1/projects/%d", projectID), updateReq, otherToken)
		// 권한 체크로 인해 403 Forbidden 또는 404 Not Found 응답을 받아야 함
		// (프로젝트를 찾지 못하거나 권한이 없으면 404)
		assert.True(t, w.Code == http.StatusForbidden || w.Code == http.StatusNotFound,
			"Expected 403 or 404, got %d", w.Code)

		// Other 사용자가 프로젝트 삭제 시도 (실패해야 함)
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/projects/%d", projectID), nil, otherToken)
		// 권한 체크로 인해 403 Forbidden 또는 404 Not Found 응답을 받아야 함
		assert.True(t, w.Code == http.StatusForbidden || w.Code == http.StatusNotFound,
			"Expected 403 or 404, got %d", w.Code)
	})
}
