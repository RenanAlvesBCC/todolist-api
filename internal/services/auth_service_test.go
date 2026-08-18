package services

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
)

func TestAuthService_RegisterAndLogin_Success(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-auth-service")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })

	db := setupTestDB(t)
	svc := NewAuthService(repository.NewUserRepository(db))

	require.NoError(t, svc.Register("ana", "senha123"))

	token, err := svc.Login("ana", "senha123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestAuthService_Register_DuplicateUsernameFails(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-auth-service")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })

	db := setupTestDB(t)
	svc := NewAuthService(repository.NewUserRepository(db))

	require.NoError(t, svc.Register("ana", "senha123"))
	err := svc.Register("ana", "outra")
	assert.Error(t, err)
}

func TestAuthService_Login_WrongPasswordFails(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-auth-service")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })

	db := setupTestDB(t)
	svc := NewAuthService(repository.NewUserRepository(db))

	require.NoError(t, svc.Register("ana", "senha123"))
	_, err := svc.Login("ana", "errada")
	assert.EqualError(t, err, "usuário ou senha incorretos")
}

func TestAuthService_Login_UnknownUserFails(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-auth-service")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })

	db := setupTestDB(t)
	svc := NewAuthService(repository.NewUserRepository(db))

	_, err := svc.Login("ghost", "x")
	assert.EqualError(t, err, "usuário ou senha incorretos")
}
