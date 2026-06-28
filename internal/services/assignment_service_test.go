package services

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

// mockAssignmentStore implementa AssignmentStore
type mockAssignmentStore struct {
	assignments []models.ListAssignment
}

func (m *mockAssignmentStore) Assign(a *models.ListAssignment) error {
	a.ID = uint(len(m.assignments) + 1)
	m.assignments = append(m.assignments, *a)
	return nil
}
func (m *mockAssignmentStore) Unassign(taskListID, userID uint) error {
	for i, a := range m.assignments {
		if a.TaskListID == taskListID && a.UserID == userID {
			m.assignments = append(m.assignments[:i], m.assignments[i+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}
func (m *mockAssignmentStore) ListByTaskList(taskListID uint) ([]models.ListAssignment, error) {
	var result []models.ListAssignment
	for _, a := range m.assignments {
		if a.TaskListID == taskListID {
			result = append(result, a)
		}
	}
	return result, nil
}
func (m *mockAssignmentStore) IsAssigned(taskListID, userID uint) (bool, error) {
	for _, a := range m.assignments {
		if a.TaskListID == taskListID && a.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}
func (m *mockAssignmentStore) UnassignAll(taskListID uint) error {
	var kept []models.ListAssignment
	for _, a := range m.assignments {
		if a.TaskListID != taskListID {
			kept = append(kept, a)
		}
	}
	m.assignments = kept
	return nil
}

type mockWsStore struct {
	wsID    uint
	members map[uint]models.WorkspaceRole
}

func (m *mockWsStore) Create(ws *models.Workspace) error               { return nil }
func (m *mockWsStore) FindByOwner(ownerID uint) (*models.Workspace, error) {
	return &models.Workspace{Model: gorm.Model{ID: m.wsID}}, nil
}
func (m *mockWsStore) FindByID(id uint) (*models.Workspace, error) {
	return &models.Workspace{Model: gorm.Model{ID: m.wsID}}, nil
}
func (m *mockWsStore) FindByMemberUserID(userID uint) (*models.Workspace, error) {
	if _, ok := m.members[userID]; ok {
		return &models.Workspace{Model: gorm.Model{ID: m.wsID}}, nil
	}
	return nil, errors.New("not found")
}
func (m *mockWsStore) Update(ws *models.Workspace) error { return nil }
func (m *mockWsStore) AddMember(mb *models.WorkspaceMember) error { return nil }
func (m *mockWsStore) FindMember(wsID, userID uint) (*models.WorkspaceMember, error) {
	return nil, errors.New("not found")
}
func (m *mockWsStore) UpdateMemberLastSeen(wsID, userID uint) error { return nil }
func (m *mockWsStore) RemoveMember(wsID, userID uint) error         { return nil }
func (m *mockWsStore) ListMembers(wsID uint) ([]models.WorkspaceMember, error) {
	return nil, nil
}
func (m *mockWsStore) CreateInvite(invite *models.WorkspaceInvite) error { return nil }
func (m *mockWsStore) FindInviteByCode(code string) (*models.WorkspaceInvite, error) {
	return nil, errors.New("not found")
}
func (m *mockWsStore) ListInvites(wsID uint) ([]models.WorkspaceInvite, error) { return nil, nil }
func (m *mockWsStore) MarkInviteUsed(invite *models.WorkspaceInvite) error     { return nil }
func (m *mockWsStore) IsMember(wsID, userID uint) (bool, error) {
	_, ok := m.members[userID]
	return ok, nil
}
func (m *mockWsStore) GetMemberRole(wsID, userID uint) (models.WorkspaceRole, error) {
	if r, ok := m.members[userID]; ok {
		return r, nil
	}
	return "", errors.New("not found")
}

func newAssignSvc(store *mockAssignmentStore, members map[uint]models.WorkspaceRole) *AssignmentService {
	ws := &mockWsStore{wsID: 10, members: members}
	return NewAssignmentService(store, ws, &mockTaskListStore{})
}

func TestAssignmentService_Assign_Success(t *testing.T) {
	store := &mockAssignmentStore{}
	svc := newAssignSvc(store, map[uint]models.WorkspaceRole{
		1: models.RoleOwner,
		5: models.RoleEditor,
	})

	err := svc.Assign(1, 100, 5)

	require.NoError(t, err)
	assert.Len(t, store.assignments, 1)
	assert.Equal(t, uint(5), store.assignments[0].UserID)
}

func TestAssignmentService_Assign_NonManagerReturnsError(t *testing.T) {
	store := &mockAssignmentStore{}
	svc := newAssignSvc(store, map[uint]models.WorkspaceRole{
		5: models.RoleEditor,
		6: models.RoleEditor,
	})

	err := svc.Assign(5, 100, 6)

	assert.EqualError(t, err, "apenas gerentes podem atribuir mecânicos")
}

func TestAssignmentService_Assign_AlreadyAssignedReturnsError(t *testing.T) {
	store := &mockAssignmentStore{}
	svc := newAssignSvc(store, map[uint]models.WorkspaceRole{
		1: models.RoleOwner,
		5: models.RoleEditor,
	})

	require.NoError(t, svc.Assign(1, 100, 5))
	err := svc.Assign(1, 100, 5)

	assert.EqualError(t, err, "mecânico já está atribuído a este veículo")
}

func TestAssignmentService_Assign_TargetNotMemberReturnsError(t *testing.T) {
	store := &mockAssignmentStore{}
	svc := newAssignSvc(store, map[uint]models.WorkspaceRole{
		1: models.RoleOwner,
	})

	err := svc.Assign(1, 100, 99)

	assert.EqualError(t, err, "usuário não é membro do workspace")
}

func TestAssignmentService_Unassign_ManagerCanRemove(t *testing.T) {
	store := &mockAssignmentStore{}
	svc := newAssignSvc(store, map[uint]models.WorkspaceRole{
		1: models.RoleOwner,
		2: models.RoleManager,
		5: models.RoleEditor,
	})

	require.NoError(t, svc.Assign(1, 100, 5))
	err := svc.Unassign(2, 100, 5)

	require.NoError(t, err)
	assert.Empty(t, store.assignments)
}

func TestAssignmentService_Unassign_EditorCannotRemove(t *testing.T) {
	store := &mockAssignmentStore{}
	svc := newAssignSvc(store, map[uint]models.WorkspaceRole{
		1: models.RoleOwner,
		5: models.RoleEditor,
		6: models.RoleEditor,
	})

	require.NoError(t, svc.Assign(1, 100, 6))
	err := svc.Unassign(5, 100, 6)

	assert.EqualError(t, err, "apenas gerentes podem remover atribuições")
}

func TestAssignmentService_List(t *testing.T) {
	store := &mockAssignmentStore{}
	svc := newAssignSvc(store, map[uint]models.WorkspaceRole{
		1: models.RoleOwner,
		5: models.RoleEditor,
		6: models.RoleEditor,
	})

	require.NoError(t, svc.Assign(1, 100, 5))
	require.NoError(t, svc.Assign(1, 100, 6))

	list, err := svc.List(100)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}
