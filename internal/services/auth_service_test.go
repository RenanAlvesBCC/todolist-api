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

	require.NoError(t, svc.Register("ana@oficina.com", "senha123", "Ana", "Silva"))

	token, err := svc.Login("ana@oficina.com", "senha123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	user, err := repository.NewUserRepository(db).FindByUsername("ana@oficina.com")
	require.NoError(t, err)
	assert.Equal(t, "Ana", user.FirstName)
	assert.Equal(t, "Silva", user.LastName)
}

func TestAuthService_Register_DuplicateUsernameFails(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-auth-service")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })

	db := setupTestDB(t)
	svc := NewAuthService(repository.NewUserRepository(db))

	require.NoError(t, svc.Register("ana@oficina.com", "senha123", "Ana", "Silva"))
	err := svc.Register("ana@oficina.com", "outra", "Ana", "Silva")
	assert.Error(t, err)
}

func TestAuthService_Login_WrongPasswordFails(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-auth-service")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })

	db := setupTestDB(t)
	svc := NewAuthService(repository.NewUserRepository(db))

	require.NoError(t, svc.Register("ana@oficina.com", "senha123", "Ana", "Silva"))
	_, err := svc.Login("ana@oficina.com", "errada")
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
