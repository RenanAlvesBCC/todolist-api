package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func createNoteUser(t *testing.T, db *gorm.DB, username, first, last string) *models.User {
	t.Helper()
	u := &models.User{Username: username, FirstName: first, LastName: last, Password: "hash"}
	require.NoError(t, db.Create(u).Error)
	return u
}

func createNoteVehicle(t *testing.T, db *gorm.DB, userID uint) *models.TaskList {
	t.Helper()
	list := &models.TaskList{Title: "Gol", UserID: userID, Status: models.StatusEmAndamento}
	require.NoError(t, NewTaskListRepository(db).Create(list))
	return list
}

func TestVehicleNoteRepository_CreateFillsAuthorName(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVehicleNoteRepository(db)
	author := createNoteUser(t, db, "ana@oficina.com", "  Ana", "Silva ")
	list := createNoteVehicle(t, db, author.ID)

	n := &models.VehicleNote{TaskListID: list.ID, AuthorID: author.ID, Text: "Cliente pediu orçamento"}
	require.NoError(t, repo.Create(n))

	assert.NotZero(t, n.ID)
	assert.Equal(t, "Ana Silva", n.AuthorName)
	assert.False(t, n.CreatedAt.IsZero())
}

func TestVehicleNoteRepository_AuthorNameFallsBackToUsername(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVehicleNoteRepository(db)
	author := createNoteUser(t, db, "legado@oficina.com", "", "")
	list := createNoteVehicle(t, db, author.ID)

	n := &models.VehicleNote{TaskListID: list.ID, AuthorID: author.ID, Text: "Sem nome"}
	require.NoError(t, repo.Create(n))

	found, err := repo.FindByID(n.ID)
	require.NoError(t, err)
	assert.Equal(t, "legado@oficina.com", found.AuthorName)
}

func TestVehicleNoteRepository_ListNewestFirstOnlyActiveOfVehicle(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVehicleNoteRepository(db)
	author := createNoteUser(t, db, "ana@oficina.com", "Ana", "Silva")
	list := createNoteVehicle(t, db, author.ID)
	other := createNoteVehicle(t, db, author.ID)

	base := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	older := &models.VehicleNote{TaskListID: list.ID, AuthorID: author.ID, Text: "antiga", CreatedAt: base}
	newer := &models.VehicleNote{TaskListID: list.ID, AuthorID: author.ID, Text: "nova", CreatedAt: base.Add(time.Hour)}
	removed := &models.VehicleNote{TaskListID: list.ID, AuthorID: author.ID, Text: "removida", CreatedAt: base.Add(2 * time.Hour)}
	foreign := &models.VehicleNote{TaskListID: other.ID, AuthorID: author.ID, Text: "outro veículo"}
	for _, n := range []*models.VehicleNote{older, newer, removed, foreign} {
		require.NoError(t, repo.Create(n))
	}
	require.NoError(t, repo.SoftDelete(removed, author.ID))

	notes, err := repo.ListByTaskList(list.ID)
	require.NoError(t, err)
	require.Len(t, notes, 2)
	assert.Equal(t, "nova", notes[0].Text)
	assert.Equal(t, "antiga", notes[1].Text)
	assert.Equal(t, "Ana Silva", notes[0].AuthorName)
}

func TestVehicleNoteRepository_ListEmptyReturnsEmptySlice(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVehicleNoteRepository(db)

	notes, err := repo.ListByTaskList(42)
	require.NoError(t, err)
	assert.NotNil(t, notes)
	assert.Empty(t, notes)
}

func TestVehicleNoteRepository_SoftDeleteKeepsRowWithDeletedBy(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVehicleNoteRepository(db)
	author := createNoteUser(t, db, "ana@oficina.com", "Ana", "Silva")
	manager := createNoteUser(t, db, "gerente@oficina.com", "Gil", "Gerente")
	list := createNoteVehicle(t, db, author.ID)

	n := &models.VehicleNote{TaskListID: list.ID, AuthorID: author.ID, Text: "Trocar correia"}
	require.NoError(t, repo.Create(n))
	require.NoError(t, repo.SoftDelete(n, manager.ID))

	_, err := repo.FindByID(n.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var raw models.VehicleNote
	require.NoError(t, db.Unscoped().First(&raw, n.ID).Error)
	assert.True(t, raw.DeletedAt.Valid)
	require.NotNil(t, raw.DeletedBy)
	assert.Equal(t, manager.ID, *raw.DeletedBy)
	assert.Equal(t, "Trocar correia", raw.Text)
}

func TestVehicleNoteRepository_SoftDeleteTwiceReturnsNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVehicleNoteRepository(db)
	author := createNoteUser(t, db, "ana@oficina.com", "Ana", "Silva")
	list := createNoteVehicle(t, db, author.ID)

	n := &models.VehicleNote{TaskListID: list.ID, AuthorID: author.ID, Text: "x"}
	require.NoError(t, repo.Create(n))
	require.NoError(t, repo.SoftDelete(n, author.ID))

	assert.ErrorIs(t, repo.SoftDelete(n, author.ID), gorm.ErrRecordNotFound)
}

func TestVehicleNoteRepository_DatabaseErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVehicleNoteRepository(db)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	assert.Error(t, repo.Create(&models.VehicleNote{TaskListID: 1, AuthorID: 1, Text: "x"}))
	_, err = repo.ListByTaskList(1)
	assert.Error(t, err)
	assert.Error(t, repo.SoftDelete(&models.VehicleNote{ID: 1}, 1))
}
