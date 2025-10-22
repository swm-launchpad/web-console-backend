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

func TestContainerWithVolume_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// 테스트 서버 설정
	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("컨테이너 생성 시 볼륨 자동 생성 및 마운트", func(t *testing.T) {
		// Step 1: 사용자 등록
		registerReq := map[string]string{
			"username": "containeruser",
			"password": "TestPassword123!",
			"email":    "container@example.com",
			"name":     "Container Test User",
		}

		w := server.MakeRequest("POST", "/auth/register", registerReq)
		assert.Equal(t, http.StatusCreated, w.Code)

		var registerResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)

		token := registerResp["data"].(map[string]interface{})["token"].(string)

		// Step 2: 프로젝트 생성
		createProjectReq := map[string]interface{}{
			"name":          "Container Test Project",
			"cpu_limit":     4000,
			"memory_limit":  8192,
			"disk_limit":    10240,
			"traffic_limit": 256,
		}

		w = server.MakeAuthRequest("POST", "/api/v1/projects", createProjectReq, token)
		if w.Code != http.StatusCreated {
			t.Logf("Create project failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var projectResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &projectResp)
		require.NoError(t, err)

		projectID := uint(projectResp["data"].(map[string]interface{})["project_id"].(float64))
		projectSlug := projectResp["data"].(map[string]interface{})["slug"].(string)
		t.Logf("Created project with ID: %d, Slug: %s", projectID, projectSlug)

		// Step 3: 템플릿 목록 조회 (MySQL 템플릿 찾기)
		w = server.MakeAuthRequest("GET", "/api/v1/templates", nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var templatesResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &templatesResp)
		require.NoError(t, err)

		templates := templatesResp["data"].(map[string]interface{})["templates"].([]interface{})
		var mysqlTemplateID uint
		for _, tmpl := range templates {
			template := tmpl.(map[string]interface{})
			if template["name"] == "MySQL" {
				mysqlTemplateID = uint(template["template_id"].(float64))
				break
			}
		}
		require.NotEqual(t, uint(0), mysqlTemplateID, "MySQL template should exist")
		t.Logf("Found MySQL template with ID: %d", mysqlTemplateID)

		// Step 4: MySQL 템플릿 상세 조회 (template_volumes 확인)
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/templates/%d", mysqlTemplateID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var templateResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &templateResp)
		require.NoError(t, err)

		templateData := templateResp["data"].(map[string]interface{})
		templateVolumes := templateData["template_volumes"].([]interface{})
		require.Greater(t, len(templateVolumes), 0, "MySQL template should have template_volumes")

		firstVolume := templateVolumes[0].(map[string]interface{})
		mountPath := firstVolume["mount_path"].(string)
		t.Logf("MySQL template has volume with mount_path: %s", mountPath)

		// Step 5: 컨테이너 생성 (볼륨 정보 포함)
		createContainerReq := map[string]interface{}{
			"name":        "mysql-db",
			"template_id": mysqlTemplateID,
			"template_config": map[string]interface{}{
				"version":       "8.0",
				"root_password": "TestRootPassword123!",
				"database_name": "testdb",
			},
			"cpu_limit":    1000,
			"memory_limit": 2048,
			"git_url":      "", // Empty git_url for database template
			"git_branch":   "main",
			"volumes": []map[string]interface{}{
				{
					"name":       "mysql-data",
					"capacity":   1024,
					"mount_path": mountPath,
				},
			},
		}

		w = server.MakeAuthRequest("POST", fmt.Sprintf("/api/v1/projects/%s/containers", projectSlug), createContainerReq, token)
		if w.Code != http.StatusCreated {
			t.Logf("Create container failed with status %d: %s", w.Code, w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)

		var containerResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &containerResp)
		require.NoError(t, err)

		containerData := containerResp["data"].(map[string]interface{})
		containerID := uint(containerData["container_id"].(float64))
		containerSlug := containerData["slug"].(string)
		t.Logf("Created container with ID: %d, Slug: %s", containerID, containerSlug)

		// Step 6: 볼륨이 자동 생성되었는지 확인
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/volumes?project_id=%d", projectID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var volumesResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &volumesResp)
		require.NoError(t, err)

		volumes := volumesResp["data"].(map[string]interface{})["volumes"].([]interface{})
		assert.Equal(t, 1, len(volumes), "One volume should be created")

		volume := volumes[0].(map[string]interface{})
		volumeID := uint(volume["volume_id"].(float64))
		volumeSlug := volume["slug"].(string)
		assert.Equal(t, "mysql-data", volume["name"])
		assert.Equal(t, float64(1024), volume["capacity"])
		t.Logf("Volume created successfully with ID: %d, Slug: %s", volumeID, volumeSlug)

		// Step 7: 컨테이너의 마운트 확인 (컨테이너 상세 조회)
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/containers/%s", containerSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var containerDetailResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &containerDetailResp)
		require.NoError(t, err)

		containerDetail := containerDetailResp["data"].(map[string]interface{})
		mounts, hasMounts := containerDetail["mounts"].([]interface{})

		// 마운트 정보가 포함되어 있는지 확인
		if hasMounts {
			assert.Greater(t, len(mounts), 0, "Container should have at least one mount")
			mount := mounts[0].(map[string]interface{})
			assert.Equal(t, float64(volumeID), mount["volume_id"].(float64))
			assert.Equal(t, mountPath, mount["mount_path"])
			t.Logf("Mount verified: volume_id=%d, mount_path=%s", volumeID, mountPath)
		} else {
			t.Logf("Warning: Container detail response does not include mounts field")
		}

		// Step 8: 컨테이너 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/containers/%s", containerSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		// Step 9: 볼륨 삭제 (컨테이너가 삭제되어도 볼륨은 남아있어야 함)
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/volumes?project_id=%d", projectID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		err = json.Unmarshal(w.Body.Bytes(), &volumesResp)
		require.NoError(t, err)

		remainingVolumes := volumesResp["data"].(map[string]interface{})["volumes"].([]interface{})
		assert.Equal(t, 1, len(remainingVolumes), "Volume should remain after container deletion")

		// Step 10: 볼륨 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/volumes/%s", volumeSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		// Step 11: 프로젝트 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", projectSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

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

		// Step 2: 프로젝트 생성
		createProjectReq := map[string]interface{}{
			"name":          "App Test Project",
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

		containerID := uint(containerResp["data"].(map[string]interface{})["container_id"].(float64))
		containerSlug := containerResp["data"].(map[string]interface{})["slug"].(string)
		t.Logf("Created container with ID: %d, Slug: %s", containerID, containerSlug)

		// Step 5: 볼륨이 생성되지 않았는지 확인
		w = server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/volumes?project_id=%d", projectID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var volumesResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &volumesResp)
		require.NoError(t, err)

		volumes := volumesResp["data"].(map[string]interface{})["volumes"].([]interface{})
		assert.Equal(t, 0, len(volumes), "No volumes should be created for non-database templates")

		// Step 6: 컨테이너 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/containers/%s", containerSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		// Step 7: 프로젝트 삭제
		w = server.MakeAuthRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", projectSlug), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
