package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

func TestWorkspaceRepository_CreateAndFindByOwner(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceRepository(db)

	ws := &models.Workspace{Name: "Oficina do João", OwnerID: 1}
	require.NoError(t, repo.Create(ws))
	assert.NotZero(t, ws.ID)

	found, err := repo.FindByOwner(1)
	require.NoError(t, err)
	assert.Equal(t, "Oficina do João", found.Name)
}

func TestWorkspaceRepository_FindByOwnerReturnsErrorWhenNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceRepository(db)

	_, err := repo.FindByOwner(999)
	assert.Error(t, err)
}

func TestWorkspaceRepository_AddMemberAndGetRole(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceRepository(db)

	ws := &models.Workspace{Name: "Oficina", OwnerID: 1}
	require.NoError(t, repo.Create(ws))

	member := &models.WorkspaceMember{
		WorkspaceID: ws.ID,
		UserID:      2,
		Role:        models.RoleEditor,
		JoinedAt:    time.Now(),
	}
	require.NoError(t, repo.AddMember(member))

	role, err := repo.GetMemberRole(ws.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, models.RoleEditor, role)
}

func TestWorkspaceRepository_IsMember(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceRepository(db)

	ws := &models.Workspace{Name: "Oficina", OwnerID: 1}
	require.NoError(t, repo.Create(ws))
	require.NoError(t, repo.AddMember(&models.WorkspaceMember{
		WorkspaceID: ws.ID, UserID: 2, Role: models.RoleEditor, JoinedAt: time.Now(),
	}))

	isMember, err := repo.IsMember(ws.ID, 2)
	require.NoError(t, err)
	assert.True(t, isMember)

	isNotMember, err := repo.IsMember(ws.ID, 99)
	require.NoError(t, err)
	assert.False(t, isNotMember)
}

func TestWorkspaceRepository_RemoveMember(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceRepository(db)

	ws := &models.Workspace{Name: "Oficina", OwnerID: 1}
	require.NoError(t, repo.Create(ws))
	require.NoError(t, repo.AddMember(&models.WorkspaceMember{
		WorkspaceID: ws.ID, UserID: 2, Role: models.RoleEditor, JoinedAt: time.Now(),
	}))

	require.NoError(t, repo.RemoveMember(ws.ID, 2))

	isMember, err := repo.IsMember(ws.ID, 2)
	require.NoError(t, err)
	assert.False(t, isMember)
}

func TestWorkspaceRepository_InviteCreateAndFind(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceRepository(db)

	ws := &models.Workspace{Name: "Oficina", OwnerID: 1}
	require.NoError(t, repo.Create(ws))

	invite := &models.WorkspaceInvite{
		WorkspaceID: ws.ID,
		InvitedBy:   1,
		Code:        "abc-123",
		Role:        models.RoleEditor,
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
	}
	require.NoError(t, repo.CreateInvite(invite))

	found, err := repo.FindInviteByCode("abc-123")
	require.NoError(t, err)
	assert.Equal(t, ws.ID, found.WorkspaceID)
	assert.Nil(t, found.UsedAt)
}

func TestWorkspaceRepository_FindByMemberUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceRepository(db)

	ws := &models.Workspace{Name: "Oficina", OwnerID: 1}
	require.NoError(t, repo.Create(ws))
	require.NoError(t, repo.AddMember(&models.WorkspaceMember{
		WorkspaceID: ws.ID, UserID: 5, Role: models.RoleManager, JoinedAt: time.Now(),
	}))

	found, err := repo.FindByMemberUserID(5)
	require.NoError(t, err)
	assert.Equal(t, ws.ID, found.ID)
}
