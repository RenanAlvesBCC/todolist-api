package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func TestTaskListRepository_LoadsCheckupVideoDoneByName(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)
	named := createNoteUser(t, db, "ana@oficina.com", "Ana", "Silva")
	legacy := createNoteUser(t, db, "legado@oficina.com", "", "")
	wsID := uint(1)

	doneAt := time.Date(2026, 11, 4, 14, 0, 0, 0, time.UTC)
	byNamed := &models.TaskList{Title: "Gol", UserID: named.ID, WorkspaceID: &wsID, Position: 0, CheckupVideoDoneAt: &doneAt, CheckupVideoDoneBy: &named.ID}
	byLegacy := &models.TaskList{Title: "Uno", UserID: named.ID, WorkspaceID: &wsID, Position: 1, CheckupVideoDoneAt: &doneAt, CheckupVideoDoneBy: &legacy.ID}
	pending := &models.TaskList{Title: "Fox", UserID: named.ID, WorkspaceID: &wsID, Position: 2}
	for _, l := range []*models.TaskList{byNamed, byLegacy, pending} {
		require.NoError(t, repo.Create(l))
	}

	lists, total, err := repo.FindAll(named.ID, &wsID, TaskListFilter{Page: 1, Limit: 10, Search: ""})
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.NotNil(t, lists[0].CheckupVideoDoneByName)
	assert.Equal(t, "Ana Silva", *lists[0].CheckupVideoDoneByName)
	require.NotNil(t, lists[1].CheckupVideoDoneByName)
	assert.Equal(t, "legado@oficina.com", *lists[1].CheckupVideoDoneByName)
	assert.Nil(t, lists[2].CheckupVideoDoneByName)
	assert.Nil(t, lists[2].CheckupVideoDoneAt)

	byID, err := repo.FindByID(byNamed.ID)
	require.NoError(t, err)
	require.NotNil(t, byID.CheckupVideoDoneAt)
	assert.True(t, doneAt.Equal(*byID.CheckupVideoDoneAt))
	assert.Equal(t, "Ana Silva", *byID.CheckupVideoDoneByName)

	byUser, err := repo.FindByIDAndUser(byLegacy.ID, named.ID)
	require.NoError(t, err)
	assert.Equal(t, "legado@oficina.com", *byUser.CheckupVideoDoneByName)

	// Filtros com nomes de coluna sem prefixo continuam funcionando junto do subselect em users.
	filtered, total, err := repo.FindAll(named.ID, &wsID, TaskListFilter{Page: 1, Limit: 10, Search: "uno", Status: string(models.StatusAguardandoBox)})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, filtered, 1)
	assert.Equal(t, "Uno", filtered[0].Title)
}

func TestTaskListRepository_UpdateIgnoresComputedCheckupName(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)
	user := createNoteUser(t, db, "ana@oficina.com", "Ana", "Silva")
	list := createNoteVehicle(t, db, user.ID)

	fake := "Outro Nome"
	list.CheckupVideoDoneBy = &user.ID
	now := time.Now().UTC()
	list.CheckupVideoDoneAt = &now
	list.CheckupVideoDoneByName = &fake
	require.NoError(t, repo.Update(list))

	got, err := repo.FindByID(list.ID)
	require.NoError(t, err)
	assert.Equal(t, "Ana Silva", *got.CheckupVideoDoneByName)
}
