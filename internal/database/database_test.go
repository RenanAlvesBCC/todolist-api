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

// legacyTaskList é a tabela task_lists como estava em produção antes dos dados do cliente.
type legacyTaskList struct {
	ID          uint `gorm:"primarykey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Title       string         `gorm:"not null"`
	Plate       string         `gorm:"not null;default:''"`
	Customer    string         `gorm:"not null;default:''"`
	UserID      uint           `gorm:"not null"`
	WorkspaceID *uint
	Status      string `gorm:"not null;default:'aguardando_box'"`
	Position    int    `gorm:"not null;default:0"`
	DeliveredAt *time.Time
}

func (legacyTaskList) TableName() string { return "task_lists" }

func TestMigrate_ExistingVehiclesGetNewColumnDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&legacyTaskList{}))
	old := legacyTaskList{Title: "Fusca", Customer: "João", UserID: 1, Status: "em_andamento"}
	require.NoError(t, db.Create(&old).Error)

	require.NoError(t, migrate(db))
	require.NoError(t, migrate(db), "migração é idempotente")

	var raw struct {
		CustomerCPF      *string
		CustomerCEP      *string
		CustomerWhatsApp *string `gorm:"column:customer_whatsapp"`
	}
	require.NoError(t, db.Raw("SELECT customer_cpf, customer_cep, customer_whatsapp FROM task_lists WHERE id = ?", old.ID).Scan(&raw).Error)
	for name, v := range map[string]*string{"customer_cpf": raw.CustomerCPF, "customer_cep": raw.CustomerCEP, "customer_whatsapp": raw.CustomerWhatsApp} {
		require.NotNil(t, v, name+" não pode ficar NULL")
		assert.Equal(t, "", *v, name)
	}

	var priority *string
	require.NoError(t, db.Raw("SELECT priority FROM task_lists WHERE id = ?", old.ID).Scan(&priority).Error)
	require.NotNil(t, priority)
	assert.Equal(t, "normal", *priority)

	var mileage *int
	require.NoError(t, db.Raw("SELECT mileage FROM task_lists WHERE id = ?", old.ID).Scan(&mileage).Error)
	assert.Nil(t, mileage, "veículo antigo fica sem quilometragem (NULL)")

	got := loadVehicle(t, db, old.ID)
	assert.Equal(t, models.PriorityNormal, got.Priority)
	assert.Nil(t, got.Mileage)
	assert.Nil(t, got.CheckupVideoDoneAt, "vídeo de check-up começa pendente")
	assert.Nil(t, got.CheckupVideoDoneBy)
	assert.Equal(t, "Fusca", got.Title)
	assert.Equal(t, "João", got.Customer)
	assert.Equal(t, models.StatusEmAndamento, got.Status)

	fresh := &models.TaskList{Title: "Novo", UserID: 1}
	require.NoError(t, db.Create(fresh).Error)
	got = loadVehicle(t, db, fresh.ID)
	assert.Equal(t, "", got.CustomerCPF)
	assert.Equal(t, models.PriorityNormal, got.Priority)
	assert.Nil(t, got.Mileage)
}

func TestMigrate_CustomerTablesDefaultsAndUniqueness(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&legacyTaskList{}))
	old := legacyTaskList{Title: "Fusca", UserID: 1, Status: "em_andamento"}
	require.NoError(t, db.Create(&old).Error)
	require.NoError(t, migrate(db))
	require.NoError(t, migrate(db), "migração é idempotente")

	got := loadVehicle(t, db, old.ID)
	assert.Nil(t, got.CustomerID, "atendimento antigo fica sem cliente")
	assert.Nil(t, got.CustomerVehicleID)

	// Cliente local mínimo: defaults vazios, sem source_id.
	c := &models.Customer{WorkspaceID: 1, Name: "Maria"}
	require.NoError(t, db.Create(c).Error)
	var loaded models.Customer
	require.NoError(t, db.First(&loaded, c.ID).Error)
	assert.Equal(t, models.SourceLocal, loaded.Source)
	assert.Nil(t, loaded.SourceID)
	assert.Equal(t, "", loaded.CPF)
	assert.Nil(t, loaded.BirthDate)
	assert.Nil(t, loaded.SyncedAt)

	// Vários clientes sem source_id e sem CPF convivem; CPF repetido na oficina não.
	require.NoError(t, db.Create(&models.Customer{WorkspaceID: 1, Name: "João"}).Error)
	require.NoError(t, db.Create(&models.Customer{WorkspaceID: 1, Name: "Ana", CPF: "12345678909"}).Error)
	assert.Error(t, db.Create(&models.Customer{WorkspaceID: 1, Name: "Bia", CPF: "12345678909"}).Error)
	require.NoError(t, db.Create(&models.Customer{WorkspaceID: 2, Name: "Bia", CPF: "12345678909"}).Error, "outra oficina pode repetir")

	sid := "123"
	require.NoError(t, db.Create(&models.Customer{WorkspaceID: 1, Name: "X", Source: models.SourceOnMotor, SourceID: &sid}).Error)
	assert.Error(t, db.Create(&models.Customer{WorkspaceID: 1, Name: "Y", Source: models.SourceOnMotor, SourceID: &sid}).Error)

	birth := models.NewDate(time.Date(1980, 5, 3, 0, 0, 0, 0, time.UTC))
	withBirth := &models.Customer{WorkspaceID: 1, Name: "Zé", BirthDate: &birth}
	require.NoError(t, db.Create(withBirth).Error)
	var loadedBirth models.Customer
	require.NoError(t, db.First(&loadedBirth, withBirth.ID).Error)
	require.NotNil(t, loadedBirth.BirthDate)
	assert.Equal(t, "1980-05-03", loadedBirth.BirthDate.String())

	// Placa única por oficina quando preenchida; sem placa pode repetir.
	require.NoError(t, db.Create(&models.CustomerVehicle{WorkspaceID: 1, Plate: "ABC1D23"}).Error)
	assert.Error(t, db.Create(&models.CustomerVehicle{WorkspaceID: 1, Plate: "ABC1D23"}).Error)
	require.NoError(t, db.Create(&models.CustomerVehicle{WorkspaceID: 1}).Error)
	require.NoError(t, db.Create(&models.CustomerVehicle{WorkspaceID: 1}).Error)

	sr := &models.ServiceRecord{WorkspaceID: 1, SourceID: "os-1"}
	require.NoError(t, db.Create(sr).Error)
	var loadedSR models.ServiceRecord
	require.NoError(t, db.First(&loadedSR, sr.ID).Error)
	assert.Equal(t, models.SourceOnMotor, loadedSR.Source)
	assert.Equal(t, models.JSONText("[]"), loadedSR.Items)
	assert.Error(t, db.Create(&models.ServiceRecord{WorkspaceID: 1, SourceID: "os-1"}).Error)

	run := &models.SyncRun{WorkspaceID: 1, StartedAt: time.Now()}
	require.NoError(t, db.Create(run).Error)
	var loadedRun models.SyncRun
	require.NoError(t, db.First(&loadedRun, run.ID).Error)
	assert.Equal(t, models.SyncStatusRunning, loadedRun.Status)
	assert.Equal(t, models.StringList{}, loadedRun.Warnings)
	assert.Equal(t, 0, loadedRun.Skipped)

	require.NoError(t, db.Create(&models.IntegrationToken{WorkspaceID: 1, Name: "robô", TokenHash: "abc", CreatedBy: 1}).Error)
	assert.Error(t, db.Create(&models.IntegrationToken{WorkspaceID: 1, Name: "outro", TokenHash: "abc", CreatedBy: 1}).Error)
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
