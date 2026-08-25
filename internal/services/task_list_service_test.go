package services

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
)

// ---- mocks ----

type mockTaskListStore struct {
	findAllFunc         func(userID uint, workspaceID *uint, filter repository.TaskListFilter) ([]models.TaskList, int64, error)
	findByIDAndUserFunc func(id, userID uint) (*models.TaskList, error)
	findByIDFunc        func(id uint) (*models.TaskList, error)
	createFunc          func(list *models.TaskList) error
	updateFunc          func(list *models.TaskList) error
	deleteFunc          func(list *models.TaskList) error
	nextPositionFunc    func(userID uint) (int, error)
	updatePositionsFunc func(userID uint, orderedIDs []uint) error
}

func (m *mockTaskListStore) FindAll(userID uint, workspaceID *uint, filter repository.TaskListFilter) ([]models.TaskList, int64, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(userID, workspaceID, filter)
	}
	return nil, 0, nil
}
func (m *mockTaskListStore) FindByIDAndUser(id, userID uint) (*models.TaskList, error) {
	if m.findByIDAndUserFunc != nil {
		return m.findByIDAndUserFunc(id, userID)
	}
	return nil, errors.New("not found")
}
func (m *mockTaskListStore) FindByID(id uint) (*models.TaskList, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	return nil, errors.New("not found")
}
func (m *mockTaskListStore) Create(list *models.TaskList) error {
	if m.createFunc != nil {
		return m.createFunc(list)
	}
	return nil
}
func (m *mockTaskListStore) Update(list *models.TaskList) error {
	if m.updateFunc != nil {
		return m.updateFunc(list)
	}
	return nil
}
func (m *mockTaskListStore) Delete(list *models.TaskList) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(list)
	}
	return nil
}
func (m *mockTaskListStore) NextPosition(userID uint) (int, error) {
	if m.nextPositionFunc != nil {
		return m.nextPositionFunc(userID)
	}
	return 0, nil
}
func (m *mockTaskListStore) UpdatePositions(userID uint, orderedIDs []uint) error {
	if m.updatePositionsFunc != nil {
		return m.updatePositionsFunc(userID, orderedIDs)
	}
	return nil
}

type mockTaskItemStore struct {
	createFunc          func(item *models.TaskItem) error
	findByIDAndListFunc func(id, taskListID uint) (*models.TaskItem, error)
	updateFunc          func(item *models.TaskItem) error
	deleteFunc          func(item *models.TaskItem) error
	deleteAllByListFunc func(taskListID uint) error
	nextPositionFunc    func(taskListID uint) (int, error)
	updatePositionsFunc func(taskListID uint, orderedIDs []uint) error
}

func (m *mockTaskItemStore) NextPosition(taskListID uint) (int, error) {
	if m.nextPositionFunc != nil {
		return m.nextPositionFunc(taskListID)
	}
	return 0, nil
}
func (m *mockTaskItemStore) UpdatePositions(taskListID uint, orderedIDs []uint) error {
	if m.updatePositionsFunc != nil {
		return m.updatePositionsFunc(taskListID, orderedIDs)
	}
	return nil
}
func (m *mockTaskItemStore) Create(item *models.TaskItem) error {
	if m.createFunc != nil {
		return m.createFunc(item)
	}
	return nil
}
func (m *mockTaskItemStore) FindByIDAndList(id, taskListID uint) (*models.TaskItem, error) {
	if m.findByIDAndListFunc != nil {
		return m.findByIDAndListFunc(id, taskListID)
	}
	return nil, errors.New("not found")
}
func (m *mockTaskItemStore) Update(item *models.TaskItem) error {
	if m.updateFunc != nil {
		return m.updateFunc(item)
	}
	return nil
}
func (m *mockTaskItemStore) Delete(item *models.TaskItem) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(item)
	}
	return nil
}
func (m *mockTaskItemStore) DeleteAllByList(taskListID uint) error {
	if m.deleteAllByListFunc != nil {
		return m.deleteAllByListFunc(taskListID)
	}
	return nil
}

type mockWsCtxStore struct {
	workspace  *models.Workspace
	memberRole map[uint]models.WorkspaceRole // userID → role
}

func (m *mockWsCtxStore) FindByMemberUserID(userID uint) (*models.Workspace, error) {
	if m.workspace != nil {
		if _, ok := m.memberRole[userID]; ok {
			return m.workspace, nil
		}
	}
	return nil, errors.New("not found")
}
func (m *mockWsCtxStore) GetMemberRole(workspaceID, userID uint) (models.WorkspaceRole, error) {
	if role, ok := m.memberRole[userID]; ok {
		return role, nil
	}
	return "", errors.New("not found")
}
func (m *mockWsCtxStore) IsMember(workspaceID, userID uint) (bool, error) {
	_, ok := m.memberRole[userID]
	return ok, nil
}

// ---- testes ----

func TestTaskListService_CreateList_Success(t *testing.T) {
	store := &mockTaskListStore{
		nextPositionFunc: func(userID uint) (int, error) { return 0, nil },
		createFunc: func(list *models.TaskList) error {
			list.ID = 1
			return nil
		},
	}
	service := NewTaskListService(store, &mockTaskItemStore{}, nil)

	list, err := service.CreateList(1, "Compras da semana", "", "")

	require.NoError(t, err)
	assert.Equal(t, uint(1), list.ID)
	assert.Equal(t, models.StatusAguardandoBox, list.Status)
	assert.Equal(t, "", list.Plate)
	assert.Equal(t, "", list.Customer)
}

func TestTaskListService_CreateList_NormalizesPlateAndCustomer(t *testing.T) {
	store := &mockTaskListStore{
		nextPositionFunc: func(userID uint) (int, error) { return 0, nil },
		createFunc:       func(list *models.TaskList) error { return nil },
	}
	service := NewTaskListService(store, &mockTaskItemStore{}, nil)

	list, err := service.CreateList(1, "Fusca", "  abc-1d23  ", "  Ana  ")
	require.NoError(t, err)
	assert.Equal(t, "ABC-1D23", list.Plate)
	assert.Equal(t, "Ana", list.Customer)
}

func TestTaskListService_CreateList_EmptyTitleReturnsError(t *testing.T) {
	service := NewTaskListService(&mockTaskListStore{}, &mockTaskItemStore{}, nil)

	_, err := service.CreateList(1, "", "", "")

	assert.EqualError(t, err, "título é obrigatório")
}

func TestTaskListService_DeleteList_RemovesItemsBeforeList(t *testing.T) {
	var deletedItemsForList uint
	var deletedList bool

	store := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return &models.TaskList{Base: models.Base{ID: id}, UserID: 1}, nil
		},
		deleteFunc: func(list *models.TaskList) error {
			deletedList = true
			return nil
		},
	}
	itemStore := &mockTaskItemStore{
		deleteAllByListFunc: func(taskListID uint) error {
			deletedItemsForList = taskListID
			return nil
		},
	}
	service := NewTaskListService(store, itemStore, nil)

	require.NoError(t, service.DeleteList(5, 1))
	assert.Equal(t, uint(5), deletedItemsForList)
	assert.True(t, deletedList)
}

func TestTaskListService_DeleteList_NotFoundReturnsError(t *testing.T) {
	store := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return nil, errors.New("record not found")
		},
	}
	service := NewTaskListService(store, &mockTaskItemStore{}, nil)

	err := service.DeleteList(5, 1)

	assert.EqualError(t, err, "lista não encontrada")
}

func TestTaskListService_UpdateItem_Success(t *testing.T) {
	store := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return &models.TaskList{Base: models.Base{ID: id}, UserID: 1}, nil
		},
	}
	itemStore := &mockTaskItemStore{
		findByIDAndListFunc: func(id, taskListID uint) (*models.TaskItem, error) {
			return &models.TaskItem{Base: models.Base{ID: id}, TaskListID: taskListID}, nil
		},
		updateFunc: func(item *models.TaskItem) error { return nil },
	}
	service := NewTaskListService(store, itemStore, nil)

	item, err := service.UpdateItem(1, 10, 1, "Leite desnatado", true)

	require.NoError(t, err)
	assert.Equal(t, "Leite desnatado", item.Text)
	assert.True(t, item.Completed)
}

func TestTaskListService_UpdateItem_ListNotOwnedByUserReturnsError(t *testing.T) {
	store := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return &models.TaskList{Base: models.Base{ID: id}, UserID: 2}, nil
		},
	}
	service := NewTaskListService(store, &mockTaskItemStore{}, nil)

	_, err := service.UpdateItem(1, 10, 999, "Tentando editar item de outro usuário", false)

	assert.EqualError(t, err, "lista não encontrada")
}

func TestTaskListService_ReorderLists_Success(t *testing.T) {
	var receivedIDs []uint
	store := &mockTaskListStore{
		updatePositionsFunc: func(userID uint, orderedIDs []uint) error {
			receivedIDs = orderedIDs
			return nil
		},
	}
	service := NewTaskListService(store, &mockTaskItemStore{}, nil)

	err := service.ReorderLists(1, []uint{3, 1, 2})

	require.NoError(t, err)
	assert.Equal(t, []uint{3, 1, 2}, receivedIDs)
}

func TestTaskListService_ReorderLists_EmptyIDsReturnsError(t *testing.T) {
	service := NewTaskListService(&mockTaskListStore{}, &mockTaskItemStore{}, nil)

	err := service.ReorderLists(1, []uint{})

	assert.EqualError(t, err, "lista de ids vazia")
}

func TestTaskListService_ReorderItems_ListNotOwnedReturnsError(t *testing.T) {
	store := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return nil, errors.New("record not found")
		},
	}
	service := NewTaskListService(store, &mockTaskItemStore{}, nil)

	err := service.ReorderItems(1, 999, []uint{1, 2})

	assert.EqualError(t, err, "lista não encontrada")
}

// ---- Fase B: testes de status ----

func TestTaskListService_ChangeStatus_EditorValidTransition(t *testing.T) {
	wsID := uint(10)
	store := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return &models.TaskList{
				Base:        models.Base{ID: id},
				WorkspaceID: &wsID,
				Status:      models.StatusEmAndamento,
				Assignments: []models.ListAssignment{{UserID: 5}},
			}, nil
		},
		updateFunc: func(list *models.TaskList) error { return nil },
	}
	ws := &mockWsCtxStore{
		workspace:  &models.Workspace{Base: models.Base{ID: wsID}},
		memberRole: map[uint]models.WorkspaceRole{5: models.RoleEditor},
	}
	service := NewTaskListService(store, &mockTaskItemStore{}, ws)

	err := service.ChangeStatus(1, 5, models.StatusAguardandoOrcamento)

	require.NoError(t, err)
}

func TestTaskListService_ChangeStatus_EditorCanSetAnyValidStatus(t *testing.T) {
	wsID := uint(10)
	store := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return &models.TaskList{
				Base:        models.Base{ID: id},
				WorkspaceID: &wsID,
				Status:      models.StatusEmAndamento,
				Assignments: []models.ListAssignment{{UserID: 5}},
			}, nil
		},
		updateFunc: func(list *models.TaskList) error { return nil },
	}
	ws := &mockWsCtxStore{
		workspace:  &models.Workspace{Base: models.Base{ID: wsID}},
		memberRole: map[uint]models.WorkspaceRole{5: models.RoleEditor},
	}
	service := NewTaskListService(store, &mockTaskItemStore{}, ws)

	err := service.ChangeStatus(1, 5, models.StatusAguardandoRetirada)

	require.NoError(t, err)
}

func TestTaskListService_ChangeStatus_OwnerCanApprove(t *testing.T) {
	wsID := uint(10)
	store := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return &models.TaskList{
				Base:        models.Base{ID: id},
				WorkspaceID: &wsID,
				Status:      models.StatusAguardandoOrcamento,
			}, nil
		},
		updateFunc: func(list *models.TaskList) error { return nil },
	}
	ws := &mockWsCtxStore{
		workspace:  &models.Workspace{Base: models.Base{ID: wsID}},
		memberRole: map[uint]models.WorkspaceRole{1: models.RoleOwner},
	}
	service := NewTaskListService(store, &mockTaskItemStore{}, ws)

	err := service.ChangeStatus(1, 1, models.StatusAguardandoRetirada)

	require.NoError(t, err)
}

func TestTaskListService_CreateList_EditorForbidden(t *testing.T) {
	wsID := uint(10)
	ws := &mockWsCtxStore{
		workspace:  &models.Workspace{Base: models.Base{ID: wsID}},
		memberRole: map[uint]models.WorkspaceRole{5: models.RoleEditor},
	}
	service := NewTaskListService(&mockTaskListStore{}, &mockTaskItemStore{}, ws)

	_, err := service.CreateList(5, "Gol 2012", "", "")

	assert.ErrorIs(t, err, ErrNotManager)
}

func TestTaskListService_ListAll_EditorSetsAssignedFilter(t *testing.T) {
	wsID := uint(10)
	var got repository.TaskListFilter
	store := &mockTaskListStore{
		findAllFunc: func(userID uint, workspaceID *uint, filter repository.TaskListFilter) ([]models.TaskList, int64, error) {
			got = filter
			return nil, 0, nil
		},
	}
	ws := &mockWsCtxStore{
		workspace:  &models.Workspace{Base: models.Base{ID: wsID}},
		memberRole: map[uint]models.WorkspaceRole{5: models.RoleEditor},
	}
	service := NewTaskListService(store, &mockTaskItemStore{}, ws)

	_, err := service.ListAll(5, "", 1, 20, "em_andamento", false)

	require.NoError(t, err)
	require.NotNil(t, got.AssignedToID)
	assert.Equal(t, uint(5), *got.AssignedToID)
	assert.Equal(t, "em_andamento", got.Status)
}

func TestTaskListService_GetList_EditorNotAssignedReturnsNotFound(t *testing.T) {
	wsID := uint(10)
	store := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return &models.TaskList{
				Base:        models.Base{ID: id},
				WorkspaceID: &wsID,
				Assignments: []models.ListAssignment{{UserID: 99}},
			}, nil
		},
	}
	ws := &mockWsCtxStore{
		workspace:  &models.Workspace{Base: models.Base{ID: wsID}},
		memberRole: map[uint]models.WorkspaceRole{5: models.RoleEditor},
	}
	service := NewTaskListService(store, &mockTaskItemStore{}, ws)

	_, err := service.GetList(1, 5)

	assert.ErrorIs(t, err, ErrNotFound)
}

func TestTaskListService_UpdateList_Success(t *testing.T) {
	store := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return &models.TaskList{Base: models.Base{ID: id}, UserID: 1, Title: "old"}, nil
		},
		updateFunc: func(list *models.TaskList) error { return nil },
	}
	svc := NewTaskListService(store, &mockTaskItemStore{}, nil)

	list, err := svc.UpdateList(1, 1, "Civic", "abc-1234", "  Maria  ")
	require.NoError(t, err)
	assert.Equal(t, "Civic", list.Title)
	assert.Equal(t, "ABC-1234", list.Plate)
	assert.Equal(t, "Maria", list.Customer)

	_, err = svc.UpdateList(1, 1, "", "", "")
	assert.EqualError(t, err, "título é obrigatório")
}

func TestTaskListService_AddItemAndDeleteItem(t *testing.T) {
	store := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return &models.TaskList{Base: models.Base{ID: id}, UserID: 1}, nil
		},
	}
	itemStore := &mockTaskItemStore{
		nextPositionFunc: func(taskListID uint) (int, error) { return 2, nil },
		createFunc: func(item *models.TaskItem) error {
			item.ID = 10
			return nil
		},
		findByIDAndListFunc: func(id, taskListID uint) (*models.TaskItem, error) {
			return &models.TaskItem{Base: models.Base{ID: id}, TaskListID: taskListID}, nil
		},
		deleteFunc: func(item *models.TaskItem) error { return nil },
	}
	svc := NewTaskListService(store, itemStore, nil)

	item, err := svc.AddItem(1, 1, "óleo")
	require.NoError(t, err)
	assert.Equal(t, "óleo", item.Text)

	_, err = svc.AddItem(1, 1, "")
	assert.EqualError(t, err, "texto do item é obrigatório")

	require.NoError(t, svc.DeleteItem(1, 10, 1))

	itemStore.findByIDAndListFunc = func(id, taskListID uint) (*models.TaskItem, error) {
		return nil, errors.New("not found")
	}
	err = svc.DeleteItem(1, 99, 1)
	assert.EqualError(t, err, "item não encontrado")
}

func TestTaskListService_ListAll_DefaultsAndMine(t *testing.T) {
	var got repository.TaskListFilter
	store := &mockTaskListStore{
		findAllFunc: func(userID uint, workspaceID *uint, filter repository.TaskListFilter) ([]models.TaskList, int64, error) {
			got = filter
			return []models.TaskList{{Title: "Gol"}}, 1, nil
		},
	}
	svc := NewTaskListService(store, &mockTaskItemStore{}, nil)

	result, err := svc.ListAll(1, "gol", 0, 0, "", true)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.Limit)
	require.NotNil(t, got.AssignedToID)
	assert.Equal(t, uint(1), *got.AssignedToID)
	assert.Equal(t, 1, result.TotalPages)
}

func TestTaskListService_ListAll_FindAllError(t *testing.T) {
	store := &mockTaskListStore{
		findAllFunc: func(userID uint, workspaceID *uint, filter repository.TaskListFilter) ([]models.TaskList, int64, error) {
			return nil, 0, errors.New("db")
		},
	}
	svc := NewTaskListService(store, &mockTaskItemStore{}, nil)
	_, err := svc.ListAll(1, "", 1, 20, "", false)
	assert.EqualError(t, err, "db")
}

func TestTaskListService_ReorderItems_Success(t *testing.T) {
	var received []uint
	store := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return &models.TaskList{Base: models.Base{ID: id}, UserID: 1}, nil
		},
	}
	itemStore := &mockTaskItemStore{
		updatePositionsFunc: func(taskListID uint, orderedIDs []uint) error {
			received = orderedIDs
			return nil
		},
	}
	svc := NewTaskListService(store, itemStore, nil)
	require.NoError(t, svc.ReorderItems(1, 1, []uint{2, 1}))
	assert.Equal(t, []uint{2, 1}, received)

	err := svc.ReorderItems(1, 1, []uint{})
	assert.EqualError(t, err, "lista de ids vazia")
}

func TestTaskListService_ChangeStatus_PersonalOwner(t *testing.T) {
	store := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return &models.TaskList{Base: models.Base{ID: id}, UserID: 1, Status: models.StatusEmAndamento}, nil
		},
		updateFunc: func(list *models.TaskList) error { return nil },
	}
	svc := NewTaskListService(store, &mockTaskItemStore{}, nil)
	require.NoError(t, svc.ChangeStatus(1, 1, models.StatusAguardandoRetirada))
}

func TestTaskListService_ChangeStatus_AccessDenied(t *testing.T) {
	store := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return &models.TaskList{Base: models.Base{ID: id}, UserID: 1, Status: models.StatusEmAndamento}, nil
		},
	}
	svc := NewTaskListService(store, &mockTaskItemStore{}, nil)
	err := svc.ChangeStatus(1, 9, models.StatusAguardandoRetirada)
	assert.EqualError(t, err, "lista não encontrada")
}

func TestTaskListService_CreateList_WithWorkspace(t *testing.T) {
	wsID := uint(10)
	store := &mockTaskListStore{
		nextPositionFunc: func(userID uint) (int, error) { return 1, nil },
		createFunc:       func(list *models.TaskList) error { list.ID = 8; return nil },
	}
	ws := &mockWsCtxStore{
		workspace:  &models.Workspace{Base: models.Base{ID: wsID}},
		memberRole: map[uint]models.WorkspaceRole{1: models.RoleOwner},
	}
	svc := NewTaskListService(store, &mockTaskItemStore{}, ws)
	list, err := svc.CreateList(1, "Gol", "xyz-9876", "João")
	require.NoError(t, err)
	require.NotNil(t, list.WorkspaceID)
	assert.Equal(t, wsID, *list.WorkspaceID)
	assert.Equal(t, "XYZ-9876", list.Plate)
	assert.Equal(t, "João", list.Customer)
}

func TestTaskListService_UpdateItem_EmptyText(t *testing.T) {
	svc := NewTaskListService(&mockTaskListStore{}, &mockTaskItemStore{}, nil)
	_, err := svc.UpdateItem(1, 1, 1, "", false)
	assert.EqualError(t, err, "texto do item é obrigatório")
}

func TestIsValidTransition_AnyValidStatusForEditorAndManager(t *testing.T) {
	assert.True(t, isValidTransition(models.RoleEditor, models.StatusAguardandoBox, models.StatusEmDiagnostico))
	assert.True(t, isValidTransition(models.RoleEditor, models.StatusEmAndamento, models.StatusEntregue))
	assert.True(t, isValidTransition(models.RoleManager, models.StatusEmAndamento, models.StatusLavaJato))
	assert.False(t, isValidTransition(models.RoleEditor, models.StatusEmAndamento, models.TaskListStatus("invalido")))
}

