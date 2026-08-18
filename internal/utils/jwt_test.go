package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateToken_RequiresJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	token, err := GenerateToken(1, "ana")
	assert.Empty(t, token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestValidateToken_RequiresJWTSecret(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	claims, err := ValidateToken("qualquer")
	assert.Nil(t, claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestGenerateAndValidateToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-jwt-utils")
	token, err := GenerateToken(9, "bia")
	require.NoError(t, err)
	claims, err := ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, float64(9), claims["user_id"])
	assert.Equal(t, "bia", claims["username"])
	_, err = TokenExpiration(token)
	require.NoError(t, err)
}
