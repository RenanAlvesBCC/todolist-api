package services

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

type mockAccessStore struct {
	isMember bool
	role     models.WorkspaceRole
	roleErr  error
}

func (m *mockAccessStore) FindByMemberUserID(userID uint) (*models.Workspace, error) {
	return nil, errors.New("unused")
}
func (m *mockAccessStore) GetMemberRole(workspaceID, userID uint) (models.WorkspaceRole, error) {
	if m.roleErr != nil {
		return "", m.roleErr
	}
	return m.role, nil
}
func (m *mockAccessStore) IsMember(workspaceID, userID uint) (bool, error) {
	return m.isMember, nil
}

func TestCheckVehicleAccess_Owner(t *testing.T) {
	list := &models.TaskList{UserID: 1}
	assert.NoError(t, checkVehicleAccess(list, 1, nil))
}

func TestCheckVehicleAccess_EditorAssigned(t *testing.T) {
	wsID := uint(10)
	list := &models.TaskList{
		UserID:      1,
		WorkspaceID: &wsID,
		Assignments: []models.ListAssignment{{UserID: 5}},
	}
	store := &mockAccessStore{isMember: true, role: models.RoleEditor}
	assert.NoError(t, checkVehicleAccess(list, 5, store))
}

func TestCheckVehicleAccess_EditorNotAssigned(t *testing.T) {
	wsID := uint(10)
	list := &models.TaskList{UserID: 1, WorkspaceID: &wsID}
	store := &mockAccessStore{isMember: true, role: models.RoleEditor}
	assert.ErrorIs(t, checkVehicleAccess(list, 5, store), ErrNotFound)
}

func TestCheckVehicleAccess_NonMember(t *testing.T) {
	wsID := uint(10)
	list := &models.TaskList{UserID: 1, WorkspaceID: &wsID}
	store := &mockAccessStore{isMember: false}
	assert.ErrorIs(t, checkVehicleAccess(list, 99, store), ErrNotFound)
}

func TestCheckVehicleAccess_ManagerMember(t *testing.T) {
	wsID := uint(10)
	list := &models.TaskList{UserID: 1, WorkspaceID: &wsID}
	store := &mockAccessStore{isMember: true, role: models.RoleManager}
	assert.NoError(t, checkVehicleAccess(list, 2, store))
}

func TestCheckVehicleAccess_RoleLookupError(t *testing.T) {
	wsID := uint(10)
	list := &models.TaskList{UserID: 1, WorkspaceID: &wsID}
	store := &mockAccessStore{isMember: true, roleErr: errors.New("missing")}
	assert.ErrorIs(t, checkVehicleAccess(list, 2, store), ErrNotFound)
}

func TestCheckVehicleAccess_NoWorkspaceNonOwner(t *testing.T) {
	list := &models.TaskList{UserID: 1}
	assert.ErrorIs(t, checkVehicleAccess(list, 2, nil), ErrNotFound)
}

func TestIsAssigned(t *testing.T) {
	list := &models.TaskList{Assignments: []models.ListAssignment{{UserID: 5}, {UserID: 8}}}
	assert.True(t, isAssigned(list, 5))
	assert.False(t, isAssigned(list, 9))
}
