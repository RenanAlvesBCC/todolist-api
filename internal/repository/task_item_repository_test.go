package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

func TestTaskItemRepository_CreateAndFindByIDAndList(t *testing.T) {
	db := setupTestDB(t)
	listRepo := NewTaskListRepository(db)
	itemRepo := NewTaskItemRepository(db)

	list := &models.TaskList{Title: "Compras da semana", UserID: 1}
	require.NoError(t, listRepo.Create(list))

	item := &models.TaskItem{Text: "Leite", TaskListID: list.ID}
	require.NoError(t, itemRepo.Create(item))

	found, err := itemRepo.FindByIDAndList(item.ID, list.ID)
	require.NoError(t, err)
	assert.Equal(t, "Leite", found.Text)
	assert.False(t, found.Completed)
}

func TestTaskItemRepository_DeleteAllByList(t *testing.T) {
	db := setupTestDB(t)
	listRepo := NewTaskListRepository(db)
	itemRepo := NewTaskItemRepository(db)

	list := &models.TaskList{Title: "Compras da semana", UserID: 1}
	require.NoError(t, listRepo.Create(list))
	require.NoError(t, itemRepo.Create(&models.TaskItem{Text: "Leite", TaskListID: list.ID}))
	require.NoError(t, itemRepo.Create(&models.TaskItem{Text: "Pão", TaskListID: list.ID}))

	item := &models.TaskItem{Text: "Leite", TaskListID: list.ID}
	require.NoError(t, itemRepo.DeleteAllByList(list.ID))

	_, err := itemRepo.FindByIDAndList(item.ID, list.ID)
	assert.Error(t, err)
}

func TestTaskItemRepository_UpdatePositions_ReordersItems(t *testing.T) {
	db := setupTestDB(t)
	listRepo := NewTaskListRepository(db)
	itemRepo := NewTaskItemRepository(db)

	list := &models.TaskList{Title: "Compras", UserID: 1}
	require.NoError(t, listRepo.Create(list))

	leite := &models.TaskItem{Text: "Leite", TaskListID: list.ID}
	pao := &models.TaskItem{Text: "Pão", TaskListID: list.ID}
	require.NoError(t, itemRepo.Create(leite))
	require.NoError(t, itemRepo.Create(pao))

	require.NoError(t, itemRepo.UpdatePositions(list.ID, []uint{pao.ID, leite.ID}))

	found, err := listRepo.FindByIDAndUser(list.ID, 1)
	require.NoError(t, err)
	require.Len(t, found.Items, 2)
	assert.Equal(t, "Pão", found.Items[0].Text)
	assert.Equal(t, "Leite", found.Items[1].Text)
}
