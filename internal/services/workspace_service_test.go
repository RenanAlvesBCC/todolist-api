package services

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

// Mock do WorkspaceStore
type mockWorkspaceStore struct {
	workspace      *models.Workspace
	members        []models.WorkspaceMember
	invites        []models.WorkspaceInvite
	createErr      error
	findByOwnerErr error
}

func (m *mockWorkspaceStore) Create(ws *models.Workspace) error {
	if m.createErr != nil {
		return m.createErr
	}
	ws.ID = 1
	m.workspace = ws
	return nil
}
func (m *mockWorkspaceStore) FindByOwner(ownerID uint) (*models.Workspace, error) {
	if m.findByOwnerErr != nil {
		return nil, m.findByOwnerErr
	}
	if m.workspace != nil && m.workspace.OwnerID == ownerID {
		return m.workspace, nil
	}
	return nil, errors.New("not found")
}
func (m *mockWorkspaceStore) FindByID(id uint) (*models.Workspace, error) {
	if m.workspace != nil {
		return m.workspace, nil
	}
	return nil, errors.New("not found")
}
func (m *mockWorkspaceStore) FindByMemberUserID(userID uint) (*models.Workspace, error) {
	for _, mb := range m.members {
		if mb.UserID == userID {
			return m.workspace, nil
		}
	}
	return nil, errors.New("not found")
}
func (m *mockWorkspaceStore) Update(ws *models.Workspace) error { m.workspace = ws; return nil }
func (m *mockWorkspaceStore) AddMember(mb *models.WorkspaceMember) error {
	m.members = append(m.members, *mb)
	return nil
}
func (m *mockWorkspaceStore) FindMember(wsID, userID uint) (*models.WorkspaceMember, error) {
	for _, mb := range m.members {
		if mb.WorkspaceID == wsID && mb.UserID == userID {
			return &mb, nil
		}
	}
	return nil, errors.New("not found")
}
func (m *mockWorkspaceStore) UpdateMemberLastSeen(wsID, userID uint) error { return nil }
func (m *mockWorkspaceStore) RemoveMember(wsID, userID uint) error {
	for i, mb := range m.members {
		if mb.WorkspaceID == wsID && mb.UserID == userID {
			m.members = append(m.members[:i], m.members[i+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}
func (m *mockWorkspaceStore) ListMembers(wsID uint) ([]models.WorkspaceMember, error) {
	return m.members, nil
}
func (m *mockWorkspaceStore) CreateInvite(invite *models.WorkspaceInvite) error {
	invite.ID = uint(len(m.invites) + 1)
	m.invites = append(m.invites, *invite)
	return nil
}
func (m *mockWorkspaceStore) FindInviteByCode(code string) (*models.WorkspaceInvite, error) {
	for _, inv := range m.invites {
		if inv.Code == code {
			return &inv, nil
		}
	}
	return nil, errors.New("not found")
}
func (m *mockWorkspaceStore) ListInvites(wsID uint) ([]models.WorkspaceInvite, error) {
	return m.invites, nil
}
func (m *mockWorkspaceStore) MarkInviteUsed(invite *models.WorkspaceInvite) error { return nil }
func (m *mockWorkspaceStore) IsMember(wsID, userID uint) (bool, error) {
	for _, mb := range m.members {
		if mb.WorkspaceID == wsID && mb.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}
func (m *mockWorkspaceStore) GetMemberRole(wsID, userID uint) (models.WorkspaceRole, error) {
	for _, mb := range m.members {
		if mb.WorkspaceID == wsID && mb.UserID == userID {
			return mb.Role, nil
		}
	}
	return "", errors.New("not found")
}

// Testes

func TestWorkspaceService_CreateWorkspace_Success(t *testing.T) {
	store := &mockWorkspaceStore{findByOwnerErr: errors.New("not found")}
	svc := NewWorkspaceService(store)

	ws, err := svc.CreateWorkspace(1, "Oficina do João", "Descrição")

	require.NoError(t, err)
	assert.Equal(t, "Oficina do João", ws.Name)
	assert.Equal(t, uint(1), ws.OwnerID)
	// Owner deve ter sido adicionado como membro automaticamente
	assert.Len(t, store.members, 1)
	assert.Equal(t, models.RoleOwner, store.members[0].Role)
}

func TestWorkspaceService_CreateWorkspace_DuplicateReturnsError(t *testing.T) {
	store := &mockWorkspaceStore{workspace: &models.Workspace{Base: models.Base{ID: 1}, Name: "Já existe", OwnerID: 1}}
	svc := NewWorkspaceService(store)

	_, err := svc.CreateWorkspace(1, "Outra oficina", "")

	assert.EqualError(t, err, "você já possui um workspace")
}

func TestWorkspaceService_CreateWorkspace_EmptyNameReturnsError(t *testing.T) {
	store := &mockWorkspaceStore{findByOwnerErr: errors.New("not found")}
	svc := NewWorkspaceService(store)

	_, err := svc.CreateWorkspace(1, "", "")

	assert.EqualError(t, err, "nome é obrigatório")
}

func TestWorkspaceService_GenerateInvite_OwnerCanInviteEditor(t *testing.T) {
	store := &mockWorkspaceStore{workspace: &models.Workspace{Base: models.Base{ID: 1}, OwnerID: 1}}
	svc := NewWorkspaceService(store)

	code, err := svc.GenerateInvite(1, models.RoleEditor)

	require.NoError(t, err)
	assert.NotEmpty(t, code)
	assert.Len(t, store.invites, 1)
}

func TestWorkspaceService_GenerateInvite_CannotInviteAsOwner(t *testing.T) {
	store := &mockWorkspaceStore{workspace: &models.Workspace{Base: models.Base{ID: 1}, OwnerID: 1}}
	svc := NewWorkspaceService(store)

	_, err := svc.GenerateInvite(1, models.RoleOwner)

	assert.EqualError(t, err, "não é possível convidar alguém como owner")
}

func TestWorkspaceService_GenerateInvite_NonOwnerCannotInvite(t *testing.T) {
	store := &mockWorkspaceStore{
		workspace: &models.Workspace{Base: models.Base{ID: 1}, OwnerID: 1},
	}
	svc := NewWorkspaceService(store)

	// userID 99 não é owner
	_, err := svc.GenerateInvite(99, models.RoleEditor)

	assert.Error(t, err)
}

func TestWorkspaceService_AcceptInvite_Success(t *testing.T) {
	ws := &models.Workspace{Base: models.Base{ID: 1}, OwnerID: 1}
	code := "valid-code"
	expires := time.Now().Add(time.Hour)
	store := &mockWorkspaceStore{
		workspace: ws,
		invites: []models.WorkspaceInvite{{
			ID: 1, WorkspaceID: 1, Code: code,
			Role: models.RoleEditor, ExpiresAt: expires,
		}},
	}
	svc := NewWorkspaceService(store)

	result, err := svc.AcceptInvite(2, code)

	require.NoError(t, err)
	assert.Equal(t, ws.ID, result.ID)
	assert.Len(t, store.members, 1)
	assert.Equal(t, models.RoleEditor, store.members[0].Role)
}

func TestWorkspaceService_AcceptInvite_ExpiredReturnsError(t *testing.T) {
	store := &mockWorkspaceStore{
		workspace: &models.Workspace{Base: models.Base{ID: 1}},
		invites: []models.WorkspaceInvite{{
			Code: "expired", ExpiresAt: time.Now().Add(-time.Hour),
		}},
	}
	svc := NewWorkspaceService(store)

	_, err := svc.AcceptInvite(2, "expired")

	assert.EqualError(t, err, "convite expirado")
}

func TestWorkspaceService_RemoveMember_OwnerCannotBeRemoved(t *testing.T) {
	store := &mockWorkspaceStore{
		workspace: &models.Workspace{Base: models.Base{ID: 1}, OwnerID: 1},
		members: []models.WorkspaceMember{
			{WorkspaceID: 1, UserID: 1, Role: models.RoleOwner},
		},
	}
	svc := NewWorkspaceService(store)

	err := svc.RemoveMember(1, 1)

	assert.EqualError(t, err, "o dono não pode ser removido do workspace")
}

func TestWorkspaceService_RemoveMember_Success(t *testing.T) {
	store := &mockWorkspaceStore{
		workspace: &models.Workspace{Base: models.Base{ID: 1}, OwnerID: 1},
		members: []models.WorkspaceMember{
			{WorkspaceID: 1, UserID: 1, Role: models.RoleOwner},
			{WorkspaceID: 1, UserID: 2, Role: models.RoleEditor},
		},
	}
	svc := NewWorkspaceService(store)

	err := svc.RemoveMember(1, 2)

	require.NoError(t, err)
	assert.Len(t, store.members, 1)
}

func TestWorkspaceService_GetMyWorkspaceAndRole(t *testing.T) {
	store := &mockWorkspaceStore{
		workspace: &models.Workspace{Base: models.Base{ID: 1}, Name: "Oficina", OwnerID: 1},
		members:   []models.WorkspaceMember{{WorkspaceID: 1, UserID: 2, Role: models.RoleEditor}},
	}
	svc := NewWorkspaceService(store)

	ws, err := svc.GetMyWorkspace(2)
	require.NoError(t, err)
	assert.Equal(t, "Oficina", ws.Name)

	role, err := svc.GetMyRole(2)
	require.NoError(t, err)
	assert.Equal(t, models.RoleEditor, role)

	_, err = svc.GetMyWorkspace(99)
	assert.EqualError(t, err, "workspace não encontrado")
	_, err = svc.GetMyRole(99)
	assert.EqualError(t, err, "workspace não encontrado")
}

func TestWorkspaceService_UpdateWorkspace(t *testing.T) {
	store := &mockWorkspaceStore{
		workspace: &models.Workspace{Base: models.Base{ID: 1}, Name: "Velha", OwnerID: 1},
	}
	svc := NewWorkspaceService(store)

	ws, err := svc.UpdateWorkspace(1, "Nova", "desc")
	require.NoError(t, err)
	assert.Equal(t, "Nova", ws.Name)

	_, err = svc.UpdateWorkspace(1, "", "")
	assert.EqualError(t, err, "nome é obrigatório")

	_, err = svc.UpdateWorkspace(99, "x", "")
	assert.EqualError(t, err, "workspace não encontrado")
}

func TestWorkspaceService_ListMembersAndInvites(t *testing.T) {
	store := &mockWorkspaceStore{
		workspace: &models.Workspace{Base: models.Base{ID: 1}, OwnerID: 1},
		members:   []models.WorkspaceMember{{WorkspaceID: 1, UserID: 1, Role: models.RoleOwner}},
		invites:   []models.WorkspaceInvite{{Code: "abc"}},
	}
	svc := NewWorkspaceService(store)

	members, err := svc.ListMembers(1)
	require.NoError(t, err)
	assert.Len(t, members, 1)

	_, err = svc.ListMembers(99)
	assert.EqualError(t, err, "workspace não encontrado")

	invites, err := svc.ListInvites(1)
	require.NoError(t, err)
	assert.Len(t, invites, 1)

	_, err = svc.ListInvites(99)
	assert.EqualError(t, err, "apenas o dono pode ver os convites")
}

func TestWorkspaceService_GetInvitePreview(t *testing.T) {
	expires := time.Now().Add(time.Hour)
	store := &mockWorkspaceStore{
		workspace: &models.Workspace{Base: models.Base{ID: 1}, Name: "Oficina"},
		invites: []models.WorkspaceInvite{{
			WorkspaceID: 1, Code: "ok", Role: models.RoleEditor, ExpiresAt: expires,
		}},
	}
	svc := NewWorkspaceService(store)

	invite, ws, err := svc.GetInvitePreview("ok")
	require.NoError(t, err)
	assert.Equal(t, "Oficina", ws.Name)
	assert.Equal(t, models.RoleEditor, invite.Role)

	_, _, err = svc.GetInvitePreview("missing")
	assert.EqualError(t, err, "convite não encontrado")
}

func TestWorkspaceService_GetInvitePreview_UsedAndExpired(t *testing.T) {
	used := time.Now()
	store := &mockWorkspaceStore{
		workspace: &models.Workspace{Base: models.Base{ID: 1}},
		invites: []models.WorkspaceInvite{
			{Code: "used", UsedAt: &used, ExpiresAt: time.Now().Add(time.Hour)},
			{Code: "expired", ExpiresAt: time.Now().Add(-time.Hour)},
		},
	}
	svc := NewWorkspaceService(store)

	_, _, err := svc.GetInvitePreview("used")
	assert.EqualError(t, err, "convite já foi utilizado")
	_, _, err = svc.GetInvitePreview("expired")
	assert.EqualError(t, err, "convite expirado")
}

func TestWorkspaceService_AcceptInvite_AlreadyMemberAndUsed(t *testing.T) {
	used := time.Now()
	store := &mockWorkspaceStore{
		workspace: &models.Workspace{Base: models.Base{ID: 1}, OwnerID: 1},
		members:   []models.WorkspaceMember{{WorkspaceID: 1, UserID: 2, Role: models.RoleEditor}},
		invites: []models.WorkspaceInvite{
			{WorkspaceID: 1, Code: "dup", Role: models.RoleEditor, ExpiresAt: time.Now().Add(time.Hour)},
			{Code: "used", UsedAt: &used, ExpiresAt: time.Now().Add(time.Hour)},
		},
	}
	svc := NewWorkspaceService(store)

	_, err := svc.AcceptInvite(2, "dup")
	assert.EqualError(t, err, "você já é membro deste workspace")
	_, err = svc.AcceptInvite(3, "used")
	assert.EqualError(t, err, "convite já foi utilizado")
	_, err = svc.AcceptInvite(3, "missing")
	assert.EqualError(t, err, "convite não encontrado")
}

func TestWorkspaceService_RemoveMember_NotOwnerAndNotFound(t *testing.T) {
	store := &mockWorkspaceStore{
		workspace: &models.Workspace{Base: models.Base{ID: 1}, OwnerID: 1},
		members:   []models.WorkspaceMember{{WorkspaceID: 1, UserID: 1, Role: models.RoleOwner}},
	}
	svc := NewWorkspaceService(store)

	err := svc.RemoveMember(99, 2)
	assert.EqualError(t, err, "apenas o dono pode remover membros")
	err = svc.RemoveMember(1, 2)
	assert.EqualError(t, err, "membro não encontrado")
}

func TestWorkspaceService_UpdateLastSeen(t *testing.T) {
	store := &mockWorkspaceStore{
		workspace: &models.Workspace{Base: models.Base{ID: 1}, OwnerID: 1},
		members:   []models.WorkspaceMember{{WorkspaceID: 1, UserID: 1, Role: models.RoleOwner}},
	}
	svc := NewWorkspaceService(store)
	svc.UpdateLastSeen(1)
	svc.UpdateLastSeen(99)
	time.Sleep(20 * time.Millisecond)
}
