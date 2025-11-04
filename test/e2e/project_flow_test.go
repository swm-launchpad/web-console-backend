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
			"name":          "Test Project",
			"slug":          "test-project",
			"plan":          "eco",
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

		var createResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &createResp)
		require.NoError(t, err)

		assert.True(t, createResp["success"].(bool))
		projectData := createResp["data"].(map[string]interface{})
		projectID := uint(projectData["project_id"].(float64))
		projectSlug := projectData["slug"].(string)

		// 기본 필드 검증
		assert.Equal(t, "Test Project", projectData["name"])
		assert.Len(t, projectSlug, 23, "Slug should be 23 characters") // slug 길이 검증
		assert.Equal(t, "p", string(projectSlug[0]), "Slug should start with 'p'")

		// Status는 active여야 함
		assert.Equal(t, "active", projectData["status"])

		// 사용자가 요청한 리소스 제한이 적용되어야 함
		assert.Equal(t, float64(1000), projectData["cpu_limit"])
		assert.Equal(t, float64(2048), projectData["memory_limit"])
		assert.Equal(t, float64(2048), projectData["disk_limit"])
		assert.Equal(t, float64(128), projectData["traffic_limit"])

		// 생성/수정 시간 존재 확인
		assert.NotEmpty(t, projectData["created_at"])
		assert.NotEmpty(t, projectData["updated_at"])

		// Step 3: 프로젝트 목록 조회
		w = server.MakeAuthRequest("GET", "/api/v1/projects", nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var listResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &listResp)
		require.NoError(t, err)

		projects := listResp["data"].(map[string]interface{})["projects"].([]interface{})
		assert.Greater(t, len(projects), 0)

		// Step 4: 프로젝트 상세 조회 (by slug)
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/projects/%s", projectSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var getResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &getResp)
		require.NoError(t, err)

		projectDetail := getResp["data"].(map[string]interface{})
		assert.Equal(t, float64(projectID), projectDetail["project_id"])
		assert.Equal(t, "Test Project", projectDetail["name"])

		// DB에서 조회한 데이터도 사용자 입력값이 적용되어야 함
		assert.Equal(t, "eco", projectDetail["plan"])
		assert.Equal(t, "active", projectDetail["status"])
		assert.Equal(t, float64(1000), projectDetail["cpu_limit"])
		assert.Equal(t, float64(2048), projectDetail["memory_limit"])
		assert.Equal(t, float64(2048), projectDetail["disk_limit"])
		assert.Equal(t, float64(128), projectDetail["traffic_limit"])

		// Step 6: 프로젝트 수정 (Beta tier limits: CPU 1 core, Memory 2GB, Disk 3GB)
		updateProjectReq := map[string]interface{}{
			"cpu_limit":     1000, // Max in beta tier
			"memory_limit":  2048, // Max in beta tier
			"disk_limit":    3072, // Max in beta tier
			"traffic_limit": 524288,
		}

		w = server.MakeAuthRequest("PUT", fmt.Sprintf("/api/v1/projects/%s", projectSlug), updateProjectReq, token)
		if w.Code != http.StatusOK {
			t.Logf("Update project failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusOK, w.Code)

		var updateResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &updateResp)
		require.NoError(t, err)

		updatedData := updateResp["data"].(map[string]interface{})
		// Name은 업데이트 안 됨 (수정하지 않음)
		assert.Equal(t, "Test Project", updatedData["name"])
		// 리소스 제한은 업데이트됨
		assert.Equal(t, float64(1000), updatedData["cpu_limit"])    // Max in beta tier
		assert.Equal(t, float64(2048), updatedData["memory_limit"]) // Max in beta tier
		assert.Equal(t, float64(3072), updatedData["disk_limit"])   // Max in beta tier
		assert.Equal(t, float64(524288), updatedData["traffic_limit"])

		// Step 7: 프로젝트 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", projectSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var deleteResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &deleteResp)
		require.NoError(t, err)

		assert.True(t, deleteResp["success"].(bool))
		assert.Equal(t, "Project deleted successfully", deleteResp["data"].(map[string]interface{})["message"])

		// Step 8: 삭제된 프로젝트 조회 시도 (실패해야 함)
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/projects/%s", projectSlug), nil, token)
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
			"name":          "Owner Project",
			"slug":          "owner-project",
			"cpu_limit":     1000,
			"memory_limit":  2048,
			"disk_limit":    2048,
			"traffic_limit": 128,
		}
		w = server.MakeAuthRequest("POST", "/api/v1/projects", createReq, ownerToken)
		assert.Equal(t, http.StatusCreated, w.Code)
		var createResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &createResp)
		_ = uint(createResp["data"].(map[string]interface{})["project_id"].(float64))
		projectSlug := createResp["data"].(map[string]interface{})["slug"].(string)

		// Other 사용자가 프로젝트 수정 시도 (실패해야 함)
		updateReq := map[string]interface{}{
			"cpu_limit": 2000,
		}
		w = server.MakeAuthRequest("PUT", fmt.Sprintf("/api/v1/projects/%s", projectSlug), updateReq, otherToken)
		// 권한 체크로 인해 403 Forbidden 또는 404 Not Found 응답을 받아야 함
		// (프로젝트를 찾지 못하거나 권한이 없으면 404)
		assert.True(t, w.Code == http.StatusForbidden || w.Code == http.StatusNotFound,
			"Expected 403 or 404, got %d", w.Code)

		// Other 사용자가 프로젝트 삭제 시도 (실패해야 함)
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", projectSlug), nil, otherToken)
		// 권한 체크로 인해 403 Forbidden 또는 404 Not Found 응답을 받아야 함
		assert.True(t, w.Code == http.StatusForbidden || w.Code == http.StatusNotFound,
			"Expected 403 or 404, got %d", w.Code)
	})
}

func TestProjectLimit_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// 테스트 서버 설정
	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("사용자당 최대 3개 프로젝트 제한", func(t *testing.T) {
		// Step 1: 사용자 등록
		registerReq := map[string]string{
			"username": "limituser",
			"password": "TestPassword123!",
			"email":    "limit@example.com",
			"name":     "Limit Test User",
		}

		w := server.MakeRequest("POST", "/auth/register", registerReq)
		assert.Equal(t, http.StatusCreated, w.Code)

		var registerResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)

		token := registerResp["data"].(map[string]interface{})["token"].(string)

		// Step 2: 3개의 프로젝트 생성 (성공해야 함)
		projectIDs := make([]uint, 0, 3)
		for i := 1; i <= 3; i++ {
			createProjectReq := map[string]interface{}{
				"name":          fmt.Sprintf("Project %d", i),
				"cpu_limit":     1000,
				"memory_limit":  2048,
				"disk_limit":    2048,
				"traffic_limit": 128,
			}

			w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, token)
			assert.Equal(t, http.StatusCreated, w.Code, "Should be able to create project %d", i)

			var createResp map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &createResp)
			require.NoError(t, err)

			projectData := createResp["data"].(map[string]interface{})
			projectID := uint(projectData["project_id"].(float64))
			projectIDs = append(projectIDs, projectID)

			// MVP 정책 검증: 각 프로젝트가 올바르게 생성되었는지 확인
			assert.Equal(t, fmt.Sprintf("Project %d", i), projectData["name"])
			assert.Equal(t, "active", projectData["status"])
			assert.Equal(t, float64(1000), projectData["cpu_limit"])
			assert.Equal(t, float64(2048), projectData["memory_limit"])
			assert.Equal(t, float64(2048), projectData["disk_limit"])
		}

		// Step 3: 4번째 프로젝트 생성 시도 (실패해야 함 - 403 Forbidden)
		createProjectReq := map[string]interface{}{
			"name":          "Project 4 - Should Fail",
			"cpu_limit":     1000,
			"memory_limit":  2048,
			"disk_limit":    2048,
			"traffic_limit": 128,
		}

		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, token)
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should not be able to create 4th project")

		var errorResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &errorResp)
		require.NoError(t, err)

		assert.False(t, errorResp["success"].(bool))
		errorMessage := errorResp["error"].(map[string]interface{})["message"].(string)
		// 에러 메시지가 프로젝트 제한과 관련된 내용을 포함하는지 확인
		assert.True(t,
			strings.Contains(strings.ToLower(errorMessage), "limit") ||
				strings.Contains(strings.ToLower(errorMessage), "maximum") ||
				strings.Contains(strings.ToLower(errorMessage), "exceed"),
			"Error message should mention project limit, got: %s", errorMessage)

		// Step 4: 프로젝트 하나 삭제 후 다시 생성 (성공해야 함)
		// First get the slug for projectIDs[0]
		w = server.MakeAuthRequest("GET", "/api/v1/projects", nil, token)
		var projectsListResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &projectsListResp)
		projects := projectsListResp["data"].(map[string]interface{})["projects"].([]interface{})
		var firstProjectSlug string
		for _, p := range projects {
			proj := p.(map[string]interface{})
			if uint(proj["project_id"].(float64)) == projectIDs[0] {
				firstProjectSlug = proj["slug"].(string)
				break
			}
		}
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", firstProjectSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		// 다시 프로젝트 생성 시도 (이제 성공해야 함)
		createProjectReq = map[string]interface{}{
			"name":          "Project After Delete",
			"cpu_limit":     1000,
			"memory_limit":  2048,
			"disk_limit":    2048,
			"traffic_limit": 128,
		}

		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, token)
		assert.Equal(t, http.StatusCreated, w.Code, "Should be able to create project after deleting one")

		var createResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &createResp)
		require.NoError(t, err)

		projectData := createResp["data"].(map[string]interface{})
		assert.Equal(t, "Project After Delete", projectData["name"])
	})
}

func TestProjectSecurityPolicies_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// 테스트 서버 설정
	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("user_id 파라미터 보안 정책 테스트", func(t *testing.T) {
		// User 1 생성
		user1Req := map[string]string{
			"username": "user1",
			"password": "User1Pass123!",
			"email":    "user1@example.com",
		}
		w := server.MakeRequest("POST", "/auth/register", user1Req)
		assert.Equal(t, http.StatusCreated, w.Code)
		var user1Resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &user1Resp)
		user1Data := user1Resp["data"].(map[string]interface{})
		user1Token := user1Data["token"].(string)
		user1ID := uint(user1Data["user_id"].(float64))

		// User 2 생성 (현재 테스트에서는 사용되지 않지만 추후 확장을 위해 생성)
		user2Req := map[string]string{
			"username": "user2",
			"password": "User2Pass123!",
			"email":    "user2@example.com",
		}
		w = server.MakeRequest("POST", "/auth/register", user2Req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// User 1이 자신의 프로젝트 목록 조회 (성공해야 함)
		w = server.MakeAuthRequest("GET", "/api/v1/projects", nil, user1Token)
		assert.Equal(t, http.StatusOK, w.Code)

		// User 1이 자신의 user_id를 명시하여 프로젝트 목록 조회 (성공해야 함)
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/projects?user_id=%d", user1ID), nil, user1Token)
		assert.Equal(t, http.StatusOK, w.Code)

		// User 1이 다른 사용자(user 2)의 프로젝트 목록 조회 시도 (실패해야 함 - 403 Forbidden)
		// 다른 사용자의 ID를 파라미터로 전달하면 권한 오류 발생
		w = server.MakeAuthRequest("GET", "/api/v1/projects?user_id=999", nil, user1Token)
		assert.Equal(t, http.StatusForbidden, w.Code,
			"Should return 403 when trying to access another user's projects")

		var forbiddenResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &forbiddenResp)
		assert.False(t, forbiddenResp["success"].(bool))

		errorMsg := forbiddenResp["error"].(map[string]interface{})["message"].(string)
		assert.True(t,
			strings.Contains(strings.ToLower(errorMsg), "permission") ||
				strings.Contains(strings.ToLower(errorMsg), "denied") ||
				strings.Contains(strings.ToLower(errorMsg), "forbidden"),
			"Error message should mention permission denied, got: %s", errorMsg)
	})
}
