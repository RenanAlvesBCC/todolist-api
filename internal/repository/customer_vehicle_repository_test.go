package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func TestCustomerRepository_VehicleFinders(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomerRepository(db)
	v := &models.CustomerVehicle{WorkspaceID: 1, Plate: "ABC1D23", Source: models.SourceOnMotor, SourceID: strPtr("v1")}
	require.NoError(t, repo.CreateVehicle(v))

	got, err := repo.FindVehicle(1, v.ID)
	require.NoError(t, err)
	assert.Equal(t, "ABC1D23", got.Plate)
	_, err = repo.FindVehicle(2, v.ID)
	assert.Error(t, err)

	byPlate, err := repo.FindVehicleByPlate(1, "ABC1D23")
	require.NoError(t, err)
	assert.Equal(t, v.ID, byPlate.ID)
	byPlate, err = repo.FindVehicleByPlate(2, "ABC1D23")
	require.NoError(t, err)
	assert.Nil(t, byPlate)

	bySource, err := repo.FindVehicleBySource(1, models.SourceOnMotor, "v1")
	require.NoError(t, err)
	assert.Equal(t, v.ID, bySource.ID)
	bySource, err = repo.FindVehicleBySource(1, models.SourceOnMotor, "v2")
	require.NoError(t, err)
	assert.Nil(t, bySource)

	v.Model = "Gol"
	require.NoError(t, repo.SaveVehicle(v))
	got, _ = repo.FindVehicle(1, v.ID)
	assert.Equal(t, "Gol", got.Model)
}

func TestCustomerRepository_HistorySources(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomerRepository(db)
	v := &models.CustomerVehicle{WorkspaceID: 1, Plate: "ABC1D23"}
	require.NoError(t, repo.CreateVehicle(v))
	older := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, db.Create(&models.ServiceRecord{WorkspaceID: 1, SourceID: "1", CustomerVehicleID: &v.ID, OpenedAt: &older}).Error)
	require.NoError(t, db.Create(&models.ServiceRecord{WorkspaceID: 1, SourceID: "2", CustomerVehicleID: &v.ID, OpenedAt: &newer}).Error)
	require.NoError(t, db.Create(&models.ServiceRecord{WorkspaceID: 2, SourceID: "3", CustomerVehicleID: &v.ID}).Error)

	records, err := repo.ServiceRecordsByVehicle(1, v.ID)
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, "2", records[0].SourceID)

	ws := uint(1)
	list := &models.TaskList{Title: "Gol", UserID: 1, WorkspaceID: &ws, CustomerVehicleID: &v.ID}
	require.NoError(t, db.Create(list).Error)
	require.NoError(t, db.Create(&models.TaskItem{TaskListID: list.ID, Text: "Troca de óleo", Position: 1}).Error)
	require.NoError(t, db.Create(&models.TaskItem{TaskListID: list.ID, Text: "Filtro", Position: 0}).Error)
	gone := &models.TaskList{Title: "Excluído", UserID: 1, WorkspaceID: &ws, CustomerVehicleID: &v.ID}
	require.NoError(t, db.Create(gone).Error)
	require.NoError(t, db.Delete(gone).Error)

	lists, err := repo.TaskListsByCustomerVehicle(1, v.ID)
	require.NoError(t, err)
	require.Len(t, lists, 1)
	require.Len(t, lists[0].Items, 2)
	assert.Equal(t, "Filtro", lists[0].Items[0].Text)
	other, err := repo.TaskListsByCustomerVehicle(2, v.ID)
	require.NoError(t, err)
	assert.Empty(t, other)
}
