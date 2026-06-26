package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

func TestQuoteRepository_CreateAndList(t *testing.T) {
	db := setupTestDB(t)
	listRepo := NewTaskListRepository(db)
	repo := NewQuoteRepository(db)

	list := &models.TaskList{Title: "Fusca 1980", UserID: 1, Status: models.StatusEmAndamento}
	require.NoError(t, listRepo.Create(list))

	q1 := &models.QuoteItem{TaskListID: list.ID, SubmittedBy: 1, Text: "Pastilha R$120"}
	q2 := &models.QuoteItem{TaskListID: list.ID, SubmittedBy: 1, Text: "Óleo R$80"}
	require.NoError(t, repo.Create(q1))
	require.NoError(t, repo.Create(q2))

	items, err := repo.ListByTaskList(list.ID)
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestQuoteRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	listRepo := NewTaskListRepository(db)
	repo := NewQuoteRepository(db)

	list := &models.TaskList{Title: "Fusca", UserID: 1, Status: models.StatusEmAndamento}
	require.NoError(t, listRepo.Create(list))

	q := &models.QuoteItem{TaskListID: list.ID, SubmittedBy: 1, Text: "Filtro R$30"}
	require.NoError(t, repo.Create(q))

	require.NoError(t, repo.Delete(q))

	items, err := repo.ListByTaskList(list.ID)
	require.NoError(t, err)
	assert.Empty(t, items)
}
