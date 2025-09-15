package password

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestNewService(t *testing.T) {
	t.Run("성공: Password 서비스 생성", func(t *testing.T) {
		service := NewService()

		assert.NotNil(t, service)
	})
}

func TestService_HashPassword(t *testing.T) {
	service := NewService()

	t.Run("성공: 일반 비밀번호 해싱", func(t *testing.T) {
		password := "MySecretPassword123!"

		hash, err := service.HashPassword(password)

		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash)
		// bcrypt 해시는 $2a$ 또는 $2b$로 시작
		assert.True(t, strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$"))
	})

	t.Run("성공: 동일한 비밀번호도 다른 해시 생성", func(t *testing.T) {
		password := "SamePassword123"

		hash1, err1 := service.HashPassword(password)
		hash2, err2 := service.HashPassword(password)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, hash1, hash2) // salt 때문에 다른 해시 생성
	})

	t.Run("성공: 최소 길이 비밀번호 해싱", func(t *testing.T) {
		password := "12345678" // 정확히 8자

		hash, err := service.HashPassword(password)

		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("성공: 72바이트 이내 긴 비밀번호 해싱", func(t *testing.T) {
		password := strings.Repeat("a", 69) + "A1" // 71자 (72바이트 이내)

		hash, err := service.HashPassword(password)

		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("실패: 72바이트 초과 비밀번호", func(t *testing.T) {
		password := strings.Repeat("a", 73) // 73자 (bcrypt 제한 초과)

		hash, err := service.HashPassword(password)

		assert.Error(t, err)
		assert.Empty(t, hash)
		assert.Contains(t, err.Error(), "bcrypt: password length exceeds 72 bytes")
	})

	t.Run("성공: 특수문자 포함 비밀번호", func(t *testing.T) {
		password := "P@$$w0rd!#%&*()[]{}|"

		hash, err := service.HashPassword(password)

		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("성공: 유니코드 문자 포함 비밀번호", func(t *testing.T) {
		password := "Pass123!한글😀"

		hash, err := service.HashPassword(password)

		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("실패: 짧은 비밀번호", func(t *testing.T) {
		passwords := []string{
			"",       // 빈 문자열
			"1",      // 1자
			"1234",   // 4자
			"1234567", // 7자
		}

		for _, password := range passwords {
			hash, err := service.HashPassword(password)

			assert.Error(t, err, "Password: %s", password)
			assert.Empty(t, hash)
			assert.Contains(t, err.Error(), "password is too weak")
		}
	})

	t.Run("성공: 단순한 비밀번호도 8자 이상이면 허용", func(t *testing.T) {
		// 실제 구현은 8자 이상만 체크하고 복잡도는 체크하지 않음
		simplePasswords := []string{
			"password",
			"12345678",
			"abcdefgh",
			"ABCDEFGH",
		}

		for _, password := range simplePasswords {
			hash, err := service.HashPassword(password)

			assert.NoError(t, err, "Password: %s", password)
			assert.NotEmpty(t, hash)
		}
	})
}

func TestService_VerifyPassword(t *testing.T) {
	service := NewService()

	t.Run("성공: 올바른 비밀번호 검증", func(t *testing.T) {
		password := "MySecretPassword123!"
		hash, err := service.HashPassword(password)
		require.NoError(t, err)

		err = service.VerifyPassword(hash, password)

		assert.NoError(t, err)
	})

	t.Run("성공: 특수문자 포함 비밀번호 검증", func(t *testing.T) {
		password := "P@$$w0rd!#%&*()"
		hash, err := service.HashPassword(password)
		require.NoError(t, err)

		err = service.VerifyPassword(hash, password)

		assert.NoError(t, err)
	})

	t.Run("성공: 유니코드 포함 비밀번호 검증", func(t *testing.T) {
		password := "Pass123!한글😀"
		hash, err := service.HashPassword(password)
		require.NoError(t, err)

		err = service.VerifyPassword(hash, password)

		assert.NoError(t, err)
	})

	t.Run("실패: 잘못된 비밀번호", func(t *testing.T) {
		password := "MySecretPassword123!"
		wrongPassword := "WrongPassword123!"
		hash, err := service.HashPassword(password)
		require.NoError(t, err)

		err = service.VerifyPassword(hash, wrongPassword)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password does not match")
	})

	t.Run("실패: 빈 비밀번호로 검증", func(t *testing.T) {
		password := "MySecretPassword123!"
		hash, err := service.HashPassword(password)
		require.NoError(t, err)

		err = service.VerifyPassword(hash, "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password does not match")
	})

	t.Run("실패: 빈 해시로 검증", func(t *testing.T) {
		password := "MySecretPassword123!"

		err := service.VerifyPassword("", password)

		assert.Error(t, err)
	})

	t.Run("실패: 잘못된 형식의 해시", func(t *testing.T) {
		password := "MySecretPassword123!"
		invalidHashes := []string{
			"not-a-bcrypt-hash",
			"$2a$",
			"$2a$10$",
			"$2a$10$invalid",
		}

		for _, hash := range invalidHashes {
			err := service.VerifyPassword(hash, password)

			assert.Error(t, err, "Hash: %s", hash)
		}
	})

	t.Run("실패: 다른 알고리즘의 해시", func(t *testing.T) {
		password := "MySecretPassword123!"
		// MD5 해시 예시
		md5Hash := "5f4dcc3b5aa765d61d8327deb882cf99"

		err := service.VerifyPassword(md5Hash, password)

		assert.Error(t, err)
	})

	t.Run("실패: 대소문자 구분", func(t *testing.T) {
		password := "MySecretPassword123!"
		wrongCase := "mysecretpassword123!"
		hash, err := service.HashPassword(password)
		require.NoError(t, err)

		err = service.VerifyPassword(hash, wrongCase)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password does not match")
	})

	t.Run("실패: 공백 차이", func(t *testing.T) {
		password := "MySecretPassword123!"
		withSpace := "MySecretPassword123! "
		hash, err := service.HashPassword(password)
		require.NoError(t, err)

		err = service.VerifyPassword(hash, withSpace)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password does not match")
	})
}

func TestBcryptCompatibility(t *testing.T) {
	service := NewService()

	t.Run("bcrypt 라이브러리와 호환성", func(t *testing.T) {
		password := "TestPassword123!"

		// 서비스로 해시 생성
		hash, err := service.HashPassword(password)
		require.NoError(t, err)

		// bcrypt 라이브러리로 직접 검증
		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		assert.NoError(t, err)
	})

	t.Run("bcrypt로 생성한 해시 검증", func(t *testing.T) {
		password := "TestPassword123!"

		// bcrypt로 직접 해시 생성
		bcryptHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		require.NoError(t, err)

		// 서비스로 검증
		err = service.VerifyPassword(string(bcryptHash), password)
		assert.NoError(t, err)
	})
}