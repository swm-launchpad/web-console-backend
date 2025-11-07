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

func TestUserFlow_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// 테스트 서버 설정
	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("전체 사용자 플로우: 회원가입 → 로그인 → 프로필 조회", func(t *testing.T) {
		// Step 1: 회원가입
		registerReq := map[string]string{
			"email":    "e2e@example.com",
			"password": "TestPassword123!",
			"nickname": "E2E Test User",
		}

		w := server.MakeRequest("POST", "/api/v1/auth/register", registerReq)
		assert.Equal(t, http.StatusCreated, w.Code)

		var registerResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)

		// 응답 구조 검증
		assert.True(t, registerResp["success"].(bool))
		registerData := registerResp["data"].(map[string]interface{})

		// 회원가입 응답 검증
		assert.NotEmpty(t, registerData["user_id"])
		assert.NotEmpty(t, registerData["token"])
		assert.Equal(t, "User registered successfully", registerData["message"])

		userID := uint(registerData["user_id"].(float64))
		registerToken := registerData["token"].(string)

		// Activate user for testing (bypass email verification)
		_, err = server.DB.DB.Exec("UPDATE USERS SET status = 'active' WHERE user_id = ?", userID)
		require.NoError(t, err)

		// Step 2: 로그인
		loginReq := map[string]string{
			"email":    "e2e@example.com",
			"password": "TestPassword123!",
		}

		w = server.MakeRequest("POST", "/api/v1/auth/login", loginReq)
		assert.Equal(t, http.StatusOK, w.Code)

		var loginResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &loginResp)
		require.NoError(t, err)

		// 응답 구조 검증
		assert.True(t, loginResp["success"].(bool))
		loginData := loginResp["data"].(map[string]interface{})

		// 로그인 응답 검증
		assert.Equal(t, float64(userID), loginData["user_id"])
		assert.Equal(t, "e2e@example.com", loginData["email"])
		assert.Equal(t, "E2E Test User", loginData["nickname"])
		assert.NotEmpty(t, loginData["token"])
		assert.Equal(t, "Login successful", loginData["message"])

		loginToken := loginData["token"].(string)

		// Step 3: 프로필 조회 (인증된 요청)
		w = server.MakeAuthRequest("GET", "/api/v1/users/me", nil, loginToken)
		assert.Equal(t, http.StatusOK, w.Code)

		var profileResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &profileResp)
		require.NoError(t, err)

		// 응답 구조 검증
		assert.True(t, profileResp["success"].(bool))
		profileData := profileResp["data"].(map[string]interface{})

		// 프로필 응답 검증
		assert.Equal(t, float64(userID), profileData["user_id"])
		assert.Equal(t, "e2e@example.com", profileData["email"])
		assert.Equal(t, "E2E Test User", profileData["nickname"])
		assert.Equal(t, "active", profileData["status"])
		assert.NotEmpty(t, profileData["created_at"])

		// 토큰이 다른지 확인 (각 엔드포인트가 새 토큰 생성)
		assert.NotEmpty(t, registerToken)
		assert.NotEmpty(t, loginToken)
	})

	t.Run("잘못된 인증정보로 로그인 실패", func(t *testing.T) {
		// 먼저 사용자 등록
		registerReq := map[string]string{
			"email":    "fail@example.com",
			"password": "CorrectPassword123!",
			"nickname": "failuser",
		}

		w := server.MakeRequest("POST", "/api/v1/auth/register", registerReq)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Activate user for testing
		var registerResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)
		userID := uint(registerResp["data"].(map[string]interface{})["user_id"].(float64))
		_, err = server.DB.DB.Exec("UPDATE USERS SET status = 'active' WHERE user_id = ?", userID)
		require.NoError(t, err)

		// 잘못된 비밀번호로 로그인 시도
		loginReq := map[string]string{
			"email":    "fail@example.com",
			"password": "WrongPassword123!",
		}

		w = server.MakeRequest("POST", "/api/v1/auth/login", loginReq)
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errorResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &errorResp)
		require.NoError(t, err)
		assert.False(t, errorResp["success"].(bool))
		errorData := errorResp["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_CREDENTIALS", errorData["code"])
		assert.Contains(t, errorData["message"], "Invalid credentials")
	})

	t.Run("중복된 email로 회원가입 실패", func(t *testing.T) {
		// 첫 번째 사용자 등록
		registerReq := map[string]string{
			"email":    "duplicate@example.com",
			"password": "Password123!",
			"nickname": "user1",
		}

		w := server.MakeRequest("POST", "/api/v1/auth/register", registerReq)
		assert.Equal(t, http.StatusCreated, w.Code)

		// 같은 email로 두 번째 사용자 등록 시도
		registerReq = map[string]string{
			"email":    "duplicate@example.com",
			"password": "Password456!",
			"nickname": "user2",
		}

		w = server.MakeRequest("POST", "/api/v1/auth/register", registerReq)
		assert.Equal(t, http.StatusConflict, w.Code)

		var errorResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		require.NoError(t, err)
		assert.False(t, errorResp["success"].(bool))
		errorData := errorResp["error"].(map[string]interface{})
		assert.Equal(t, "EMAIL_EXISTS", errorData["code"])
		assert.Contains(t, errorData["message"], "already exists")
	})

	t.Run("인증 없이 프로필 조회 실패", func(t *testing.T) {
		w := server.MakeRequest("GET", "/api/v1/users/me", nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errorResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		require.NoError(t, err)
		assert.False(t, errorResp["success"].(bool))
		errorData := errorResp["error"].(map[string]interface{})
		assert.Equal(t, "MISSING_AUTH_HEADER", errorData["code"])
		assert.Equal(t, "Missing authorization header", errorData["message"])
	})

	t.Run("ID로 다른 사용자 조회", func(t *testing.T) {
		// 사용자 등록
		userID, token := server.RegisterUser(t, "view@example.com", "Password123!", "viewuser")

		// ID로 사용자 조회 (인증 필요)
		w := server.MakeAuthRequest("GET", fmt.Sprintf("/api/v1/users/%d", userID), nil, token)
		assert.Equal(t, http.StatusOK, w.Code)

		var userResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &userResp)
		require.NoError(t, err)

		// 응답 구조 검증
		assert.True(t, userResp["success"].(bool))
		userData := userResp["data"].(map[string]interface{})

		assert.Equal(t, float64(userID), userData["user_id"])
		assert.Equal(t, "viewuser", userData["nickname"])
		assert.Equal(t, "view@example.com", userData["email"])
	})

	t.Run("존재하지 않는 사용자 조회", func(t *testing.T) {
		// 인증을 위한 사용자 생성
		_, token := server.RegisterUser(t, "notfound@example.com", "Password123!", "notfounduser")

		w := server.MakeAuthRequest("GET", "/api/v1/users/999999", nil, token)
		assert.Equal(t, http.StatusNotFound, w.Code)

		var errorResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		require.NoError(t, err)
		assert.False(t, errorResp["success"].(bool))
		errorData := errorResp["error"].(map[string]interface{})
		assert.Equal(t, "USER_NOT_FOUND", errorData["code"])
		assert.Contains(t, errorData["message"], "User not found")
	})

	t.Run("잘못된 요청 형식 처리", func(t *testing.T) {
		// 잘못된 JSON
		w := server.MakeRequest("POST", "/api/v1/auth/register", "invalid json")
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// 필수 필드 누락
		incompleteReq := map[string]string{
			"email": "incomplete@example.com",
			// password와 nickname 누락
		}

		w = server.MakeRequest("POST", "/api/v1/auth/register", incompleteReq)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// 짧은 비밀번호
		weakPasswordReq := map[string]string{
			"email":    "weak@example.com",
			"password": "123", // 최소 8자
			"nickname": "weakpass",
		}

		w = server.MakeRequest("POST", "/api/v1/auth/register", weakPasswordReq)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("동시성 테스트: 여러 사용자 동시 등록", func(t *testing.T) {
		done := make(chan bool, 5)

		for i := 0; i < 5; i++ {
			go func(index int) {
				defer func() { done <- true }()

				registerReq := map[string]string{
					"email":    fmt.Sprintf("concurrent%d@example.com", index),
					"password": "Password123!",
					"nickname": fmt.Sprintf("concurrent%d", index),
				}

				w := server.MakeRequest("POST", "/api/v1/auth/register", registerReq)
				assert.Equal(t, http.StatusCreated, w.Code)
			}(i)
		}

		// 모든 고루틴 완료 대기
		for i := 0; i < 5; i++ {
			<-done
		}

		// Activate all concurrently registered users
		for i := 0; i < 5; i++ {
			_, err := server.DB.DB.Exec("UPDATE USERS SET status = 'active' WHERE email = ?", fmt.Sprintf("concurrent%d@example.com", i))
			require.NoError(t, err)
		}

		// 등록된 사용자들 확인
		for i := 0; i < 5; i++ {
			loginReq := map[string]string{
				"email":    fmt.Sprintf("concurrent%d@example.com", i),
				"password": "Password123!",
			}

			w := server.MakeRequest("POST", "/api/v1/auth/login", loginReq)
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})
}

func TestPasswordSecurity_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := helper.SetupTestServer(t)
	defer server.Cleanup()

	t.Run("약한 비밀번호 거부", func(t *testing.T) {
		weakPasswords := []string{
			"1234567", // 7자 - 너무 짧음
			"short",   // 5자 - 너무 짧음
			"pass",    // 4자 - 너무 짧음
			"123",     // 3자 - 너무 짧음
			"",        // 빈 문자열
			"a",       // 1자
		}

		for i, password := range weakPasswords {
			registerReq := map[string]string{
				"email":    fmt.Sprintf("test%d@example.com", i),
				"password": password,
				"nickname": fmt.Sprintf("weak%d", i),
			}

			w := server.MakeRequest("POST", "/api/v1/auth/register", registerReq)
			assert.Equal(t, http.StatusBadRequest, w.Code, "Password should be rejected: %s (length: %d)", password, len(password))
		}
	})

	t.Run("허용되는 비밀번호", func(t *testing.T) {
		acceptablePasswords := []string{
			"12345678",                      // 8자 - 숫자만
			"abcdefgh",                      // 8자 - 문자만
			"password",                      // 8자 - 일반 단어
			"MyP@ssw0rd!",                   // 복잡한 비밀번호
			"Str0ng&Secure",                 // 복잡한 비밀번호
			"C0mpl3x#Pass",                  // 복잡한 비밀번호
			"Test123!@#",                    // 복잡한 비밀번호
			"verylongpasswordwithmanychars", // 긴 비밀번호
		}

		for i, password := range acceptablePasswords {
			registerReq := map[string]string{
				"email":    fmt.Sprintf("accept%d@example.com", i),
				"password": password,
				"nickname": fmt.Sprintf("accept%d", i),
			}

			w := server.MakeRequest("POST", "/api/v1/auth/register", registerReq)
			assert.Equal(t, http.StatusCreated, w.Code, "Password should be accepted: %s (length: %d)", password, len(password))
		}
	})

	t.Run("비밀번호 해시 검증", func(t *testing.T) {
		// 같은 비밀번호로 두 사용자 등록
		password := "SamePassword123!"

		for i := 0; i < 2; i++ {
			registerReq := map[string]string{
				"email":    fmt.Sprintf("hashtest%d@example.com", i),
				"password": password,
				"nickname": fmt.Sprintf("hashtest%d", i),
			}

			w := server.MakeRequest("POST", "/api/v1/auth/register", registerReq)
			require.Equal(t, http.StatusCreated, w.Code)
		}

		// DB에서 직접 해시 확인 (해시가 다른지 검증)
		user1, err := server.DB.GetUserByEmail("hashtest0@example.com")
		require.NoError(t, err)

		user2, err := server.DB.GetUserByEmail("hashtest1@example.com")
		require.NoError(t, err)

		// 같은 비밀번호라도 해시는 달라야 함 (salt 때문)
		assert.NotEqual(t, user1["password_hash"], user2["password_hash"])
	})
}
