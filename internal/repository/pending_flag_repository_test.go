package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

func TestPendingFlagRepository_CreateAndList(t *testing.T) {
	db := setupTestDB(t)
	listRepo := NewTaskListRepository(db)
	repo := NewPendingFlagRepository(db)

	list := &models.TaskList{Title: "Gol 2005", UserID: 1, Status: models.StatusEmAndamento}
	require.NoError(t, listRepo.Create(list))

	f1 := &models.PendingFlag{TaskListID: list.ID, CreatedBy: 1, FlagType: models.FlagAguardandoCliente}
	f2 := &models.PendingFlag{TaskListID: list.ID, CreatedBy: 1, FlagType: models.FlagProcurandoPeca}
	require.NoError(t, repo.Create(f1))
	require.NoError(t, repo.Create(f2))

	flags, err := repo.ListByTaskList(list.ID)
	require.NoError(t, err)
	assert.Len(t, flags, 2)
}

func TestPendingFlagRepository_Resolve(t *testing.T) {
	db := setupTestDB(t)
	listRepo := NewTaskListRepository(db)
	repo := NewPendingFlagRepository(db)

	list := &models.TaskList{Title: "Gol", UserID: 1, Status: models.StatusEmAndamento}
	require.NoError(t, listRepo.Create(list))

	f := &models.PendingFlag{TaskListID: list.ID, CreatedBy: 1, FlagType: models.FlagAguardandoML}
	require.NoError(t, repo.Create(f))

	now := time.Now()
	userID := uint(1)
	f.ResolvedAt = &now
	f.ResolvedBy = &userID
	require.NoError(t, repo.Update(f))

	found, err := repo.FindByID(f.ID)
	require.NoError(t, err)
	assert.NotNil(t, found.ResolvedAt)
}
