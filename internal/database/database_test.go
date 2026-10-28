package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&models.TaskList{}, &models.TaskItem{}, &models.ListAssignment{}))
	return db
}

func createVehicle(t *testing.T, db *gorm.DB, status models.TaskListStatus, updatedAt time.Time, deliveredAt *time.Time) uint {
	t.Helper()
	list := &models.TaskList{Title: "Veículo", UserID: 1, Status: status, DeliveredAt: deliveredAt}
	require.NoError(t, db.Create(list).Error)
	require.NoError(t, db.Model(list).UpdateColumn("updated_at", updatedAt).Error)
	return list.ID
}

func loadVehicle(t *testing.T, db *gorm.DB, id uint) models.TaskList {
	t.Helper()
	var list models.TaskList
	require.NoError(t, db.First(&list, id).Error)
	return list
}

func TestBackfillDeliveredAt_FillsOnlyEntregueWithoutDate(t *testing.T) {
	db := setupTestDB(t)
	oldUpdate := time.Date(2026, 9, 10, 15, 0, 0, 0, time.UTC)
	existing := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)

	missing := createVehicle(t, db, models.StatusEntregue, oldUpdate, nil)
	kept := createVehicle(t, db, models.StatusEntregue, oldUpdate, &existing)
	notDelivered := createVehicle(t, db, models.StatusEmAndamento, oldUpdate, nil)

	require.NoError(t, backfillDeliveredAt(db))
	require.NoError(t, backfillDeliveredAt(db))

	got := loadVehicle(t, db, missing)
	require.NotNil(t, got.DeliveredAt)
	assert.True(t, oldUpdate.Equal(*got.DeliveredAt))
	assert.True(t, oldUpdate.Equal(got.UpdatedAt), "backfill não deve tocar updated_at")

	got = loadVehicle(t, db, kept)
	require.NotNil(t, got.DeliveredAt)
	assert.True(t, existing.Equal(*got.DeliveredAt))

	assert.Nil(t, loadVehicle(t, db, notDelivered).DeliveredAt)
}

func TestMigrateLegacyStatuses_ConcluidoKeepsOriginalUpdatedAtAsDeliveredAt(t *testing.T) {
	db := setupTestDB(t)
	oldUpdate := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)

	concluido := createVehicle(t, db, models.TaskListStatus("concluido"), oldUpdate, nil)
	aprovado := createVehicle(t, db, models.TaskListStatus("aprovado"), oldUpdate, nil)

	migrateLegacyStatuses(db)
	require.NoError(t, backfillDeliveredAt(db))

	got := loadVehicle(t, db, concluido)
	assert.Equal(t, models.StatusEntregue, got.Status)
	require.NotNil(t, got.DeliveredAt)
	assert.True(t, oldUpdate.Equal(*got.DeliveredAt))

	got = loadVehicle(t, db, aprovado)
	assert.Equal(t, models.StatusAguardandoRetirada, got.Status)
	assert.Nil(t, got.DeliveredAt)
}
