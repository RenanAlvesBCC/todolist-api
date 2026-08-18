package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
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

func TestWorkspaceRepository_UpdateFindMemberListAndInvites(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceRepository(db)

	ws := &models.Workspace{Name: "Oficina", Description: "a", OwnerID: 1}
	require.NoError(t, repo.Create(ws))

	ws.Name = "Oficina Centro"
	require.NoError(t, repo.Update(ws))
	byID, err := repo.FindByID(ws.ID)
	require.NoError(t, err)
	assert.Equal(t, "Oficina Centro", byID.Name)

	member := &models.WorkspaceMember{WorkspaceID: ws.ID, UserID: 2, Role: models.RoleEditor, JoinedAt: time.Now()}
	require.NoError(t, repo.AddMember(member))

	foundMember, err := repo.FindMember(ws.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, models.RoleEditor, foundMember.Role)

	_, err = repo.FindMember(ws.ID, 99)
	assert.Error(t, err)

	require.NoError(t, repo.UpdateMemberLastSeen(ws.ID, 2))

	members, err := repo.ListMembers(ws.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)

	invite := &models.WorkspaceInvite{
		WorkspaceID: ws.ID, InvitedBy: 1, Code: "code-1", Role: models.RoleEditor, ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.CreateInvite(invite))

	invites, err := repo.ListInvites(ws.ID)
	require.NoError(t, err)
	assert.Len(t, invites, 1)

	now := time.Now()
	uid := uint(2)
	invite.UsedAt = &now
	invite.UsedBy = &uid
	require.NoError(t, repo.MarkInviteUsed(invite))

	used, err := repo.FindInviteByCode("code-1")
	require.NoError(t, err)
	assert.NotNil(t, used.UsedAt)

	_, err = repo.FindInviteByCode("missing")
	assert.Error(t, err)

	_, err = repo.GetMemberRole(ws.ID, 99)
	assert.Error(t, err)

	_, err = repo.FindByMemberUserID(99)
	assert.Error(t, err)
}

func TestWorkspaceRepository_FindByIDUpdateAndMembers(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceRepository(db)

	ws := &models.Workspace{Name: "Oficina", OwnerID: 1, Description: "a"}
	require.NoError(t, repo.Create(ws))

	found, err := repo.FindByID(ws.ID)
	require.NoError(t, err)
	assert.Equal(t, "Oficina", found.Name)

	_, err = repo.FindByID(999)
	assert.Error(t, err)

	ws.Name = "Oficina Nova"
	require.NoError(t, repo.Update(ws))
	updated, err := repo.FindByID(ws.ID)
	require.NoError(t, err)
	assert.Equal(t, "Oficina Nova", updated.Name)

	member := &models.WorkspaceMember{WorkspaceID: ws.ID, UserID: 2, Role: models.RoleEditor, JoinedAt: time.Now()}
	require.NoError(t, repo.AddMember(member))

	got, err := repo.FindMember(ws.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, models.RoleEditor, got.Role)

	_, err = repo.FindMember(ws.ID, 99)
	assert.Error(t, err)

	require.NoError(t, repo.UpdateMemberLastSeen(ws.ID, 2))
	members, err := repo.ListMembers(ws.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.NotNil(t, members[0].LastSeenAt)

	_, err = repo.FindByMemberUserID(99)
	assert.Error(t, err)

	_, err = repo.GetMemberRole(ws.ID, 99)
	assert.Error(t, err)
}

func TestWorkspaceRepository_ListInvitesAndMarkUsed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkspaceRepository(db)

	ws := &models.Workspace{Name: "Oficina", OwnerID: 1}
	require.NoError(t, repo.Create(ws))

	invite := &models.WorkspaceInvite{
		WorkspaceID: ws.ID, InvitedBy: 1, Code: "code-1",
		Role: models.RoleEditor, ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.CreateInvite(invite))

	listed, err := repo.ListInvites(ws.ID)
	require.NoError(t, err)
	require.Len(t, listed, 1)

	_, err = repo.FindInviteByCode("missing")
	assert.Error(t, err)

	now := time.Now()
	uid := uint(2)
	invite.UsedAt = &now
	invite.UsedBy = &uid
	require.NoError(t, repo.MarkInviteUsed(invite))

	found, err := repo.FindInviteByCode("code-1")
	require.NoError(t, err)
	assert.NotNil(t, found.UsedAt)
}
