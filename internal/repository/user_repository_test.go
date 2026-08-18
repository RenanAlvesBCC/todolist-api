package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func TestUserRepository_CreateAndFindByUsername(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	user := &models.User{Username: "ana", Password: "hash"}
	require.NoError(t, repo.Create(user))
	assert.NotZero(t, user.ID)

	found, err := repo.FindByUsername("ana")
	require.NoError(t, err)
	assert.Equal(t, "ana", found.Username)
	assert.Equal(t, "hash", found.Password)
}

func TestUserRepository_FindByUsername_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	_, err := repo.FindByUsername("ghost")
	assert.Error(t, err)
}

func TestUserRepository_Create_DuplicateUsername(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	require.NoError(t, repo.Create(&models.User{Username: "ana", Password: "a"}))
	err := repo.Create(&models.User{Username: "ana", Password: "b"})
	assert.Error(t, err)
}
