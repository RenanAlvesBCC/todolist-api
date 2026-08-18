package services

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
)

type mockAuditLogStore struct {
	lastFilter repository.AuditFilter
	rows       []repository.AuditLogRow
	total      int64
	err        error
}

func (m *mockAuditLogStore) ListLogs(filter repository.AuditFilter) ([]repository.AuditLogRow, int64, error) {
	m.lastFilter = filter
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.rows, m.total, nil
}

type mockAuditWorkspaceStore struct {
	ws      *models.Workspace
	wsErr   error
	role    models.WorkspaceRole
	roleErr error
}

func (m *mockAuditWorkspaceStore) FindByMemberUserID(userID uint) (*models.Workspace, error) {
	if m.wsErr != nil {
		return nil, m.wsErr
	}
	return m.ws, nil
}
func (m *mockAuditWorkspaceStore) GetMemberRole(workspaceID, userID uint) (models.WorkspaceRole, error) {
	if m.roleErr != nil {
		return "", m.roleErr
	}
	return m.role, nil
}

func TestAuditService_List_EditorForbidden(t *testing.T) {
	svc := NewAuditService(&mockAuditLogStore{}, &mockAuditWorkspaceStore{
		ws:   &models.Workspace{Base: models.Base{ID: 1}},
		role: models.RoleEditor,
	})

	_, err := svc.List(5, "", nil, nil, nil, 1, 20)
	assert.EqualError(t, err, "apenas gerentes podem ver a auditoria")
}

func TestAuditService_List_WorkspaceNotFound(t *testing.T) {
	svc := NewAuditService(&mockAuditLogStore{}, &mockAuditWorkspaceStore{
		wsErr: errors.New("not found"),
	})

	_, err := svc.List(1, "", nil, nil, nil, 1, 20)
	assert.EqualError(t, err, "workspace não encontrado")
}

func TestAuditService_List_DefaultPagination(t *testing.T) {
	store := &mockAuditLogStore{rows: []repository.AuditLogRow{{ID: 1}}, total: 1}
	svc := NewAuditService(store, &mockAuditWorkspaceStore{
		ws:   &models.Workspace{Base: models.Base{ID: 1}},
		role: models.RoleOwner,
	})

	result, err := svc.List(1, "acesso", nil, nil, nil, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 1, store.lastFilter.Page)
	assert.Equal(t, 20, store.lastFilter.Limit)
	assert.Equal(t, "acesso", store.lastFilter.Category)
}

func TestAuditService_List_LimitAboveMaxDefaults(t *testing.T) {
	store := &mockAuditLogStore{}
	svc := NewAuditService(store, &mockAuditWorkspaceStore{
		ws:   &models.Workspace{Base: models.Base{ID: 1}},
		role: models.RoleManager,
	})

	_, err := svc.List(1, "", nil, nil, nil, 2, 500)
	require.NoError(t, err)
	assert.Equal(t, 2, store.lastFilter.Page)
	assert.Equal(t, 20, store.lastFilter.Limit)
}

func TestAuditService_List_Success(t *testing.T) {
	actor := uint(9)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	store := &mockAuditLogStore{
		rows:  []repository.AuditLogRow{{ID: 1, Action: "login_success", Category: "acesso"}},
		total: 1,
	}
	svc := NewAuditService(store, &mockAuditWorkspaceStore{
		ws:   &models.Workspace{Base: models.Base{ID: 1}},
		role: models.RoleManager,
	})

	result, err := svc.List(1, "acesso", &actor, &from, &to, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 1, result.TotalPages)
	require.Len(t, result.Logs, 1)
	assert.Equal(t, &actor, store.lastFilter.ActorID)
	assert.Equal(t, &from, store.lastFilter.DateFrom)
	assert.Equal(t, &to, store.lastFilter.DateTo)
}

func TestAuditService_List_RepoError(t *testing.T) {
	svc := NewAuditService(&mockAuditLogStore{err: errors.New("db")}, &mockAuditWorkspaceStore{
		ws:   &models.Workspace{Base: models.Base{ID: 1}},
		role: models.RoleOwner,
	})

	_, err := svc.List(1, "", nil, nil, nil, 1, 20)
	assert.EqualError(t, err, "db")
}

func TestAuditService_List_RoleLookupErrorForbidden(t *testing.T) {
	svc := NewAuditService(&mockAuditLogStore{}, &mockAuditWorkspaceStore{
		ws:      &models.Workspace{Base: models.Base{ID: 1}},
		roleErr: errors.New("missing"),
	})

	_, err := svc.List(1, "", nil, nil, nil, 1, 20)
	assert.EqualError(t, err, "apenas gerentes podem ver a auditoria")
}
