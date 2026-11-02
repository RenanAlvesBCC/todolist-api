package repository

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func TestTaskListRepository_PreloadsActiveNotesNewestFirst(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)
	notes := NewVehicleNoteRepository(db)
	author := createNoteUser(t, db, "ana@oficina.com", "Ana", "Silva")
	legacy := createNoteUser(t, db, "legado@oficina.com", "", "")

	wsID := uint(1)
	first := &models.TaskList{Title: "Gol", UserID: author.ID, WorkspaceID: &wsID, Position: 0}
	second := &models.TaskList{Title: "Uno", UserID: author.ID, WorkspaceID: &wsID, Position: 1}
	require.NoError(t, repo.Create(first))
	require.NoError(t, repo.Create(second))

	base := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	older := &models.VehicleNote{TaskListID: first.ID, AuthorID: author.ID, Text: "antiga", CreatedAt: base}
	newer := &models.VehicleNote{TaskListID: first.ID, AuthorID: legacy.ID, Text: "nova", CreatedAt: base.Add(time.Hour)}
	removed := &models.VehicleNote{TaskListID: first.ID, AuthorID: author.ID, Text: "removida", CreatedAt: base.Add(2 * time.Hour)}
	for _, n := range []*models.VehicleNote{older, newer, removed} {
		require.NoError(t, notes.Create(n))
	}
	require.NoError(t, notes.SoftDelete(removed, author.ID))

	lists, total, err := repo.FindAll(author.ID, &wsID, TaskListFilter{Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, lists, 2)
	assert.Equal(t, first.ID, lists[0].ID, "ordem dos veículos segue position")
	require.Len(t, lists[0].Notes, 2)
	assert.Equal(t, "nova", lists[0].Notes[0].Text)
	assert.Equal(t, "legado@oficina.com", lists[0].Notes[0].AuthorName)
	assert.Equal(t, "antiga", lists[0].Notes[1].Text)
	assert.Equal(t, "Ana Silva", lists[0].Notes[1].AuthorName)
	assert.NotNil(t, lists[1].Notes)
	assert.Empty(t, lists[1].Notes)

	byID, err := repo.FindByID(first.ID)
	require.NoError(t, err)
	require.Len(t, byID.Notes, 2)
	assert.Equal(t, "nova", byID.Notes[0].Text)

	byUser, err := repo.FindByIDAndUser(first.ID, author.ID)
	require.NoError(t, err)
	assert.Len(t, byUser.Notes, 2)

	raw, err := json.Marshal(byID)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))
	jsonNotes, ok := body["notes"].([]any)
	require.True(t, ok)
	require.Len(t, jsonNotes, 2)
	note := jsonNotes[0].(map[string]any)
	assert.ElementsMatch(t,
		[]string{"id", "task_list_id", "author_id", "author_name", "text", "created_at"},
		keysOf(note))
}

func TestTaskListRepository_UpdateDoesNotTouchNotes(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskListRepository(db)
	notes := NewVehicleNoteRepository(db)
	author := createNoteUser(t, db, "ana@oficina.com", "Ana", "Silva")
	list := createNoteVehicle(t, db, author.ID)

	n := &models.VehicleNote{TaskListID: list.ID, AuthorID: author.ID, Text: "original"}
	require.NoError(t, notes.Create(n))

	loaded, err := repo.FindByID(list.ID)
	require.NoError(t, err)
	require.Len(t, loaded.Notes, 1)
	require.NoError(t, notes.SoftDelete(n, author.ID))

	loaded.Title = "Gol G5"
	loaded.Notes[0].Text = "alterada"
	require.NoError(t, repo.Update(loaded))

	var raw models.VehicleNote
	require.NoError(t, db.Unscoped().First(&raw, n.ID).Error)
	assert.Equal(t, "original", raw.Text)
	assert.True(t, raw.DeletedAt.Valid, "nota excluída continua excluída")

	reloaded, err := repo.FindByID(list.ID)
	require.NoError(t, err)
	assert.Equal(t, "Gol G5", reloaded.Title)
	assert.Empty(t, reloaded.Notes)
}

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
