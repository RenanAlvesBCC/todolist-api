package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func TestTaskListRepository_CreateAndFindByIDAndUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)

	list := &models.TaskList{Title: "Compras da semana", UserID: 1}
	require.NoError(t, repo.Create(list))
	assert.NotZero(t, list.ID)

	found, err := repo.FindByIDAndUser(list.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "Compras da semana", found.Title)
}

func TestTaskListRepository_FindByIDAndUser_WrongUserReturnsError(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)

	list := &models.TaskList{Title: "Trabalho", UserID: 1}
	require.NoError(t, repo.Create(list))

	_, err := repo.FindByIDAndUser(list.ID, 999)
	assert.Error(t, err)
}

func TestTaskListRepository_FindAllByUser_FiltersAndPaginates(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)

	require.NoError(t, repo.Create(&models.TaskList{Title: "Compras da semana", UserID: 1}))
	require.NoError(t, repo.Create(&models.TaskList{Title: "Trabalho", UserID: 1}))
	require.NoError(t, repo.Create(&models.TaskList{Title: "Compras do mês", UserID: 1}))
	require.NoError(t, repo.Create(&models.TaskList{Title: "Lista de outro usuário", UserID: 2}))

	lists, total, err := repo.FindAll(1, nil, TaskListFilter{Search: "compras", Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, lists, 2)
}

func TestTaskListRepository_FindAllByUser_SearchMatchesPlateOrCustomer(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)

	require.NoError(t, repo.Create(&models.TaskList{Title: "Gol", Plate: "ABC1D23", Customer: "Maria", UserID: 1}))
	require.NoError(t, repo.Create(&models.TaskList{Title: "Civic", Plate: "XYZ9A87", Customer: "João", UserID: 1}))
	require.NoError(t, repo.Create(&models.TaskList{Title: "Uno", Plate: "QWE0F00", Customer: "Pedro", UserID: 1}))

	byPlate, totalPlate, err := repo.FindAll(1, nil, TaskListFilter{Search: "ABC1", Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), totalPlate)
	require.Len(t, byPlate, 1)
	assert.Equal(t, "Gol", byPlate[0].Title)

	byCustomer, totalCustomer, err := repo.FindAll(1, nil, TaskListFilter{Search: "João", Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), totalCustomer)
	require.Len(t, byCustomer, 1)
	assert.Equal(t, "Civic", byCustomer[0].Title)
}

func TestTaskListRepository_FindAllByUser_PreloadsItems(t *testing.T) {
	db := setupTestDB(t)
	listRepo := NewTaskListRepository(db)
	itemRepo := NewTaskItemRepository(db)

	list := &models.TaskList{Title: "Compras da semana", UserID: 1}
	require.NoError(t, listRepo.Create(list))
	require.NoError(t, itemRepo.Create(&models.TaskItem{Text: "Leite", TaskListID: list.ID}))

	lists, _, err := listRepo.FindAll(1, nil, TaskListFilter{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Len(t, lists, 1)
	assert.Len(t, lists[0].Items, 1)
	assert.Equal(t, "Leite", lists[0].Items[0].Text)
}

func TestTaskListRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)

	list := &models.TaskList{Title: "Casa", UserID: 1}
	require.NoError(t, repo.Create(list))
	require.NoError(t, repo.Delete(list))

	_, err := repo.FindByIDAndUser(list.ID, 1)
	assert.Error(t, err)
}

func TestTaskListRepository_NextPosition(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)

	require.NoError(t, repo.Create(&models.TaskList{Title: "Compras", UserID: 1}))

	position, err := repo.NextPosition(1)
	require.NoError(t, err)
	assert.Equal(t, 1, position)
}

func TestTaskListRepository_UpdatePositions_ReordersOnlyOwnedLists(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)

	a := &models.TaskList{Title: "A", UserID: 1}
	b := &models.TaskList{Title: "B", UserID: 1}
	require.NoError(t, repo.Create(a))
	require.NoError(t, repo.Create(b))

	require.NoError(t, repo.UpdatePositions(1, []uint{b.ID, a.ID}))

	lists, _, err := repo.FindAll(1, nil, TaskListFilter{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Len(t, lists, 2)
	assert.Equal(t, "B", lists[0].Title)
	assert.Equal(t, "A", lists[1].Title)
}

func TestTaskListRepository_FindByIDAndUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)

	list := &models.TaskList{Title: "Fusca", UserID: 1, Status: models.StatusEmAndamento}
	require.NoError(t, repo.Create(list))

	found, err := repo.FindByID(list.ID)
	require.NoError(t, err)
	assert.Equal(t, "Fusca", found.Title)

	_, err = repo.FindByID(999)
	assert.Error(t, err)

	found.Title = "Fusca 1980"
	found.Status = models.StatusAguardandoPeca
	require.NoError(t, repo.Update(found))

	updated, err := repo.FindByID(list.ID)
	require.NoError(t, err)
	assert.Equal(t, "Fusca 1980", updated.Title)
	assert.Equal(t, models.StatusAguardandoPeca, updated.Status)
}

func TestTaskListRepository_FindAll_StatusAndAssignedFilters(t *testing.T) {
	db := setupTestDB(t)
	listRepo := NewTaskListRepository(db)
	assignRepo := NewListAssignmentRepository(db)

	wsID := uint(10)
	a := &models.TaskList{Title: "A", UserID: 1, WorkspaceID: &wsID, Status: models.StatusEmAndamento}
	b := &models.TaskList{Title: "B", UserID: 1, WorkspaceID: &wsID, Status: models.StatusEntregue}
	require.NoError(t, listRepo.Create(a))
	require.NoError(t, listRepo.Create(b))
	require.NoError(t, assignRepo.Assign(&models.ListAssignment{TaskListID: a.ID, UserID: 5, AssignedBy: 1, AssignedAt: time.Now()}))

	byStatus, total, err := listRepo.FindAll(1, &wsID, TaskListFilter{Status: string(models.StatusEntregue), Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, byStatus, 1)
	assert.Equal(t, "B", byStatus[0].Title)

	assignedTo := uint(5)
	mine, total, err := listRepo.FindAll(1, &wsID, TaskListFilter{AssignedToID: &assignedTo, Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, mine, 1)
	assert.Equal(t, "A", mine[0].Title)
}

func TestTaskListRepository_FindAll_WorkspaceStatusAndAssigned(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)
	assignRepo := NewListAssignmentRepository(db)

	wsID := uint(10)
	a := &models.TaskList{Title: "Gol", UserID: 1, WorkspaceID: &wsID, Status: models.StatusEmAndamento}
	b := &models.TaskList{Title: "Uno", UserID: 1, WorkspaceID: &wsID, Status: models.StatusAguardandoRetirada}
	personal := &models.TaskList{Title: "Pessoal", UserID: 1}
	require.NoError(t, repo.Create(a))
	require.NoError(t, repo.Create(b))
	require.NoError(t, repo.Create(personal))
	require.NoError(t, assignRepo.Assign(&models.ListAssignment{TaskListID: a.ID, UserID: 5, AssignedBy: 1, AssignedAt: time.Now()}))

	all, total, err := repo.FindAll(1, &wsID, TaskListFilter{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, all, 3)

	byStatus, stTotal, err := repo.FindAll(1, &wsID, TaskListFilter{Status: string(models.StatusAguardandoRetirada), Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), stTotal)
	require.Len(t, byStatus, 1)
	assert.Equal(t, "Uno", byStatus[0].Title)

	assignedTo := uint(5)
	mine, mineTotal, err := repo.FindAll(1, &wsID, TaskListFilter{AssignedToID: &assignedTo, Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), mineTotal)
	require.Len(t, mine, 1)
	assert.Equal(t, "Gol", mine[0].Title)
}

func TestTaskListRepository_FindAll_AssignedIncludesLegacyNilWorkspace(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)
	assignRepo := NewListAssignmentRepository(db)

	wsID := uint(10)
	legacy := &models.TaskList{Title: "Fiesta legado", UserID: 1, WorkspaceID: nil, Status: models.StatusEmAndamento}
	require.NoError(t, repo.Create(legacy))
	require.NoError(t, assignRepo.Assign(&models.ListAssignment{
		TaskListID: legacy.ID, UserID: 5, AssignedBy: 1, AssignedAt: time.Now(),
	}))

	assignedTo := uint(5)
	mine, total, err := repo.FindAll(5, &wsID, TaskListFilter{AssignedToID: &assignedTo, Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, mine, 1)
	assert.Equal(t, "Fiesta legado", mine[0].Title)
}
