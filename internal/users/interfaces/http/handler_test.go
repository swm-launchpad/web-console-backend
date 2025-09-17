package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthHandler_RequestValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Register: 잘못된 JSON 형식", func(t *testing.T) {
		handler := NewAuthHandler(nil, nil)
		router := gin.New()
		router.POST("/auth/register", handler.Register)

		body := []byte(`{"username": "test", "password": 123}`) // password should be string
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid request format")
	})

	t.Run("Register: 필수 필드 누락", func(t *testing.T) {
		handler := NewAuthHandler(nil, nil)
		router := gin.New()
		router.POST("/auth/register", handler.Register)

		testCases := []struct {
			name string
			body map[string]string
		}{
			{
				name: "username 누락",
				body: map[string]string{
					"password": "password123",
					"email":    "test@example.com",
				},
			},
			{
				name: "password 누락",
				body: map[string]string{
					"username": "testuser",
					"email":    "test@example.com",
				},
			},
			{
				name: "email 누락",
				body: map[string]string{
					"username": "testuser",
					"password": "password123",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				body, _ := json.Marshal(tc.body)
				req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "Invalid request format")
			})
		}
	})

	t.Run("Register: 최소 길이 검증", func(t *testing.T) {
		handler := NewAuthHandler(nil, nil)
		router := gin.New()
		router.POST("/auth/register", handler.Register)

		testCases := []struct {
			name string
			body RegisterRequest
		}{
			{
				name: "짧은 username",
				body: RegisterRequest{
					Username: "ab", // min 3
					Password: "password123",
					Email:    "test@example.com",
				},
			},
			{
				name: "짧은 password",
				body: RegisterRequest{
					Username: "testuser",
					Password: "1234567", // min 8
					Email:    "test@example.com",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				body, _ := json.Marshal(tc.body)
				req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "Invalid request format")
			})
		}
	})

	t.Run("Register: 이메일 형식 검증", func(t *testing.T) {
		handler := NewAuthHandler(nil, nil)
		router := gin.New()
		router.POST("/auth/register", handler.Register)

		invalidEmails := []string{
			"invalid-email",
			"@example.com",
			"test@",
			"test",
			"test@.com",
			"test@example",
		}

		for _, email := range invalidEmails {
			t.Run(email, func(t *testing.T) {
				reqBody := RegisterRequest{
					Username: "testuser",
					Password: "password123",
					Email:    email,
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "Invalid request format")
			})
		}
	})

	t.Run("Login: 필수 필드 누락", func(t *testing.T) {
		handler := NewAuthHandler(nil, nil)
		router := gin.New()
		router.POST("/auth/login", handler.Login)

		testCases := []struct {
			name string
			body map[string]string
		}{
			{
				name: "username 누락",
				body: map[string]string{
					"password": "password123",
				},
			},
			{
				name: "password 누락",
				body: map[string]string{
					"username": "testuser",
				},
			},
			{
				name: "모든 필드 누락",
				body: map[string]string{},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				body, _ := json.Marshal(tc.body)
				req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "Invalid request format")
			})
		}
	})
}

func TestUserHandler_RequestValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("GetCurrentUser: 인증되지 않은 사용자", func(t *testing.T) {
		handler := NewUserHandler(nil)
		router := gin.New()
		router.GET("/users/me", handler.GetCurrentUser) // No userID in context

		req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "User not authenticated", response["error"])
	})

	t.Run("GetUserByID: 잘못된 ID 형식", func(t *testing.T) {
		handler := NewUserHandler(nil)
		router := gin.New()
		router.GET("/users/:id", handler.GetUserByID)

		testCases := []struct {
			name string
			id   string
		}{
			{"문자열 ID", "abc"},
			{"특수문자 ID", "!@#"},
			{"소수 ID", "12.34"},
			{"음수 ID", "-1"},
			{"매우 큰 숫자", "18446744073709551616"}, // uint64 max + 1
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/users/"+tc.id, nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "Invalid user ID format", response["error"])
			})
		}
	})

	t.Run("GetUserByID: 빈 ID", func(t *testing.T) {
		handler := NewUserHandler(nil)
		router := gin.New()

		// 빈 파라미터를 시뮬레이션하기 위한 커스텀 핸들러
		router.GET("/users/", func(c *gin.Context) {
			c.Params = []gin.Param{} // 빈 파라미터
			handler.GetUserByID(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/users/", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "User ID is required", response["error"])
	})
}

func TestContainsFunction(t *testing.T) {
	testCases := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"포함된 경우", "hello world", "world", true},
		{"포함되지 않은 경우", "hello world", "foo", false},
		{"빈 문자열", "", "test", false},
		{"빈 서브스트링", "test", "", true},
		{"같은 문자열", "test", "test", true},
		{"시작 부분", "test string", "test", true},
		{"끝 부분", "test string", "string", true},
		{"대소문자 구분", "Hello", "hello", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := contains(tc.s, tc.substr)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestJSONResponseFormats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("날짜 형식 확인", func(t *testing.T) {
		testTime, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:45Z")

		response := UserResponse{
			UserID:    123,
			Username:  "testuser",
			Status:    "active",
			CreatedAt: testTime.Format("2006-01-02T15:04:05Z"),
		}

		jsonData, err := json.Marshal(response)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(jsonData, &parsed)
		require.NoError(t, err)

		assert.Equal(t, "2024-01-15T10:30:45Z", parsed["created_at"])
	})

	t.Run("omitempty 태그 동작 확인", func(t *testing.T) {
		// 빈 필드를 가진 응답
		response := UserResponse{
			UserID:   123,
			Username: "testuser",
			Email:    "", // omitempty
			Name:     "", // omitempty
			Phone:    "", // omitempty
			Organization: "", // omitempty
			Status:   "active",
			CreatedAt: time.Now().Format("2006-01-02T15:04:05Z"),
		}

		jsonData, err := json.Marshal(response)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(jsonData, &parsed)
		require.NoError(t, err)

		_, hasEmail := parsed["email"]
		assert.True(t, hasEmail)

		// omitempty 필드들은 JSON에 포함되지 않아야 함
		_, hasName := parsed["name"]
		_, hasPhone := parsed["phone"]
		_, hasOrg := parsed["organization"]

		assert.False(t, hasName)
		assert.False(t, hasPhone)
		assert.False(t, hasOrg)

		// 필수 필드들은 항상 포함되어야 함
		assert.NotNil(t, parsed["user_id"])
		assert.NotNil(t, parsed["username"])
		assert.NotNil(t, parsed["email"])
		assert.NotNil(t, parsed["status"])
		assert.NotNil(t, parsed["created_at"])
	})
}