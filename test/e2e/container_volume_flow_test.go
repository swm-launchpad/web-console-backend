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

func TestContainerWithoutVolume_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// 테스트 서버 설정
	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("볼륨 없이 컨테이너 생성 (일반 애플리케이션 템플릿)", func(t *testing.T) {
		// Step 1: 사용자 등록
		registerReq := map[string]string{
			"username": "appuser",
			"password": "TestPassword123!",
			"email":    "app@example.com",
		}

		w := server.MakeRequest("POST", "/auth/register", registerReq)
		assert.Equal(t, http.StatusCreated, w.Code)

		var registerResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)

		token := registerResp["data"].(map[string]interface{})["token"].(string)

		// Step 2: 프로젝트 생성 (베타 티어 제한 내로)
		createProjectReq := map[string]interface{}{
			"name":          "App Test Project",
			"cpu_limit":     1000, // 1 core (beta limit)
			"memory_limit":  2048, // 2GB (beta limit)
			"disk_limit":    3072, // 3GB (beta limit)
			"traffic_limit": 128,
		}

		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, token)
		if w.Code != http.StatusCreated {
			t.Logf("Project creation failed with code %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var projectResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &projectResp)
		require.NoError(t, err)

		projectSlug := projectResp["data"].(map[string]interface{})["slug"].(string)

		// Step 3: 템플릿 목록 조회 (Node.js 템플릿 찾기)
		w = server.MakeAuthRequest("GET", "/api/v1/templates", nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var templatesResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &templatesResp)
		require.NoError(t, err)

		templates := templatesResp["data"].(map[string]interface{})["templates"].([]interface{})
		var expressTemplateID uint
		for _, tmpl := range templates {
			template := tmpl.(map[string]interface{})
			// Use Express.js template since it doesn't have template_volumes
			if template["name"] == "Express.js" {
				expressTemplateID = uint(template["template_id"].(float64))
				break
			}
		}
		require.NotEqual(t, uint(0), expressTemplateID, "Express.js template should exist")

		// Step 4: 컨테이너 생성 (볼륨 없음)
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

		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/projects/%s/containers", projectSlug), createContainerReq, token)
		if w.Code != http.StatusCreated {
			t.Logf("Create container failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var containerResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &containerResp)
		require.NoError(t, err)

		_ = uint(containerResp["data"].(map[string]interface{})["container_id"].(float64))
		containerSlug := containerResp["data"].(map[string]interface{})["slug"].(string)
		t.Logf("Created container with Slug: %s", containerSlug)

		// Step 5: 컨테이너에 볼륨이 없는지 확인 (새 API 사용)
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/containers/%s/volumes", containerSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var volumesResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &volumesResp)
		require.NoError(t, err)

		volumes := volumesResp["data"].(map[string]interface{})["volumes"].([]interface{})
		assert.Equal(t, 0, len(volumes), "No volumes should exist for newly created container without volumes")
		t.Logf("Verified container has no volumes")

		// Step 6: 컨테이너 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/containers/%s", containerSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		// Step 7: 프로젝트 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", projectSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
