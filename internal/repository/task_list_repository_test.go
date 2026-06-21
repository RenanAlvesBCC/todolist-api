package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
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

	lists, total, err := repo.FindAllByUser(1, TaskListFilter{Search: "compras", Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, lists, 2)
}

func TestTaskListRepository_FindAllByUser_PreloadsItems(t *testing.T) {
	db := setupTestDB(t)
	listRepo := NewTaskListRepository(db)
	itemRepo := NewTaskItemRepository(db)

	list := &models.TaskList{Title: "Compras da semana", UserID: 1}
	require.NoError(t, listRepo.Create(list))
	require.NoError(t, itemRepo.Create(&models.TaskItem{Text: "Leite", TaskListID: list.ID}))

	lists, _, err := listRepo.FindAllByUser(1, TaskListFilter{Page: 1, Limit: 10})
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

	lists, _, err := repo.FindAllByUser(1, TaskListFilter{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Len(t, lists, 2)
	assert.Equal(t, "B", lists[0].Title)
	assert.Equal(t, "A", lists[1].Title)
}
