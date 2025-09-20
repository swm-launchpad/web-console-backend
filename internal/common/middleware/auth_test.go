package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jwtService "github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}

// testErrorResponse represents the error response structure for testing
type testErrorResponse struct {
	Success bool `json:"success"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestAuthMiddleware_RequireAuth(t *testing.T) {
	jwtSvc := jwtService.NewJWTUtil("test-secret-key-for-testing")
	middleware := NewAuthMiddleware(jwtSvc)

	t.Run("인증 헤더가 없는 경우", func(t *testing.T) {
		// Arrange
		router := setupTestRouter()
		router.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var errResp testErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.False(t, errResp.Success)
		assert.Equal(t, "MISSING_AUTH_HEADER", errResp.Error.Code)
		assert.Equal(t, "authorization header is required", errResp.Error.Message)
	})

	t.Run("Bearer 접두사가 없는 경우", func(t *testing.T) {
		// Arrange
		router := setupTestRouter()
		router.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "InvalidFormat token123")
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var errResp testErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.False(t, errResp.Success)
		assert.Equal(t, "INVALID_AUTH_FORMAT", errResp.Error.Code)
		assert.Equal(t, "invalid authorization header format", errResp.Error.Message)
	})

	t.Run("Bearer 뒤에 토큰이 없는 경우", func(t *testing.T) {
		// Arrange
		router := setupTestRouter()
		router.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer ")
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var errResp testErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.False(t, errResp.Success)
		assert.Equal(t, "MISSING_TOKEN", errResp.Error.Code)
		assert.Equal(t, "token is required", errResp.Error.Message)
	})

	t.Run("잘못된 토큰 형식", func(t *testing.T) {
		// Arrange
		router := setupTestRouter()
		router.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.format")
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var errResp testErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.False(t, errResp.Success)
		assert.Equal(t, "INVALID_TOKEN", errResp.Error.Code)
		assert.Equal(t, "invalid token", errResp.Error.Message)
	})

	t.Run("만료된 토큰", func(t *testing.T) {
		// Arrange
		router := setupTestRouter()
		router.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Create an expired token
		claims := jwt.MapClaims{
			"user_id": float64(123),                          // JWT numbers are float64
			"exp":     time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte("test-secret-key-for-testing"))
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var errResp testErrorResponse
		err = json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.False(t, errResp.Success)
		assert.Equal(t, "INVALID_TOKEN", errResp.Error.Code)
		assert.Equal(t, "invalid token", errResp.Error.Message)
	})

	t.Run("잘못된 시크릿으로 서명된 토큰", func(t *testing.T) {
		// Arrange
		router := setupTestRouter()
		router.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Create a token with wrong secret
		claims := jwt.MapClaims{
			"user_id": float64(123),
			"exp":     time.Now().Add(time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte("wrong-secret"))
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var errResp testErrorResponse
		err = json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.False(t, errResp.Success)
		assert.Equal(t, "INVALID_TOKEN", errResp.Error.Code)
		assert.Equal(t, "invalid token", errResp.Error.Message)
	})

	t.Run("유효한 토큰", func(t *testing.T) {
		// Arrange
		router := setupTestRouter()
		var capturedUserID uint
		var capturedAuth bool

		router.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
			capturedUserID = c.GetUint("userID")
			capturedAuth = c.GetBool("authenticated")
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Generate valid token
		ctx := context.Background()
		tokenString, err := jwtSvc.GenerateToken(ctx, 123)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, uint(123), capturedUserID)
		assert.True(t, capturedAuth)

		var successResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &successResp)
		require.NoError(t, err)
		// This is a test endpoint that returns gin.H{"message": "success"}, not using our response format
		assert.Equal(t, "success", successResp["message"])
	})
}

func TestAuthMiddleware_OptionalAuth(t *testing.T) {
	jwtSvc := jwtService.NewJWTUtil("test-secret-key-for-testing")
	middleware := NewAuthMiddleware(jwtSvc)

	t.Run("토큰이 없는 경우 - 인증되지 않은 상태로 진행", func(t *testing.T) {
		// Arrange
		router := setupTestRouter()
		var capturedAuth bool
		var capturedUserID uint

		router.GET("/optional", middleware.OptionalAuth(), func(c *gin.Context) {
			capturedAuth = c.GetBool("authenticated")
			capturedUserID = c.GetUint("userID")
			c.JSON(http.StatusOK, gin.H{"authenticated": capturedAuth})
		})

		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.False(t, capturedAuth)
		assert.Equal(t, uint(0), capturedUserID)
	})

	t.Run("잘못된 형식의 토큰 - 인증되지 않은 상태로 진행", func(t *testing.T) {
		// Arrange
		router := setupTestRouter()
		var capturedAuth bool

		router.GET("/optional", middleware.OptionalAuth(), func(c *gin.Context) {
			capturedAuth = c.GetBool("authenticated")
			c.JSON(http.StatusOK, gin.H{"authenticated": capturedAuth})
		})

		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		req.Header.Set("Authorization", "InvalidFormat token123")
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.False(t, capturedAuth)
	})

	t.Run("만료된 토큰 - 인증되지 않은 상태로 진행", func(t *testing.T) {
		// Arrange
		router := setupTestRouter()
		var capturedAuth bool

		router.GET("/optional", middleware.OptionalAuth(), func(c *gin.Context) {
			capturedAuth = c.GetBool("authenticated")
			c.JSON(http.StatusOK, gin.H{"authenticated": capturedAuth})
		})

		// Create an expired token
		claims := jwt.MapClaims{
			"user_id": float64(123),
			"exp":     time.Now().Add(-1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte("test-secret-key-for-testing"))
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.False(t, capturedAuth)
	})

	t.Run("유효한 토큰 - 인증된 상태로 진행", func(t *testing.T) {
		// Arrange
		router := setupTestRouter()
		var capturedUserID uint
		var capturedAuth bool

		router.GET("/optional", middleware.OptionalAuth(), func(c *gin.Context) {
			capturedUserID = c.GetUint("userID")
			capturedAuth = c.GetBool("authenticated")
			c.JSON(http.StatusOK, gin.H{
				"authenticated": capturedAuth,
				"userID":        capturedUserID,
			})
		})

		// Generate valid token
		ctx := context.Background()
		tokenString, err := jwtSvc.GenerateToken(ctx, 456)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, uint(456), capturedUserID)
		assert.True(t, capturedAuth)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["authenticated"].(bool))
		assert.Equal(t, float64(456), response["userID"].(float64))
	})

	t.Run("Bearer 뒤에 토큰이 없는 경우 - 인증되지 않은 상태로 진행", func(t *testing.T) {
		// Arrange
		router := setupTestRouter()
		var capturedAuth bool

		router.GET("/optional", middleware.OptionalAuth(), func(c *gin.Context) {
			capturedAuth = c.GetBool("authenticated")
			c.JSON(http.StatusOK, gin.H{"authenticated": capturedAuth})
		})

		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		req.Header.Set("Authorization", "Bearer ")
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.False(t, capturedAuth)
	})
}

func TestAuthMiddleware_Integration(t *testing.T) {
	jwtSvc := jwtService.NewJWTUtil("test-secret-key-for-testing")
	middleware := NewAuthMiddleware(jwtSvc)

	t.Run("미들웨어 체인 테스트 - RequireAuth가 요청을 차단", func(t *testing.T) {
		// Arrange
		router := setupTestRouter()
		handlerCalled := false

		router.GET("/protected",
			middleware.RequireAuth(),
			func(c *gin.Context) {
				handlerCalled = true
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.False(t, handlerCalled, "핸들러가 호출되면 안됨")
	})

	t.Run("미들웨어 체인 테스트 - OptionalAuth는 요청을 차단하지 않음", func(t *testing.T) {
		// Arrange
		router := setupTestRouter()
		handlerCalled := false

		router.GET("/optional",
			middleware.OptionalAuth(),
			func(c *gin.Context) {
				handlerCalled = true
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, handlerCalled, "핸들러가 호출되어야 함")
	})
}
