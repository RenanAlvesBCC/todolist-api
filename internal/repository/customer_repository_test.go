package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func strPtr(s string) *string { return &s }

func seedCustomer(t *testing.T, repo *CustomerRepository, c models.Customer) *models.Customer {
	t.Helper()
	require.NoError(t, repo.CreateCustomer(&c))
	return &c
}

func TestCustomerRepository_FindersAreWorkspaceScoped(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomerRepository(db)
	maria := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "Maria", CPF: "12345678909", Source: models.SourceOnMotor, SourceID: strPtr("77")})

	got, err := repo.FindCustomer(1, maria.ID)
	require.NoError(t, err)
	assert.Equal(t, "Maria", got.Name)
	_, err = repo.FindCustomer(2, maria.ID)
	assert.Error(t, err, "outra oficina não enxerga")

	byCPF, err := repo.FindCustomerByCPF(1, "12345678909")
	require.NoError(t, err)
	assert.Equal(t, maria.ID, byCPF.ID)
	byCPF, err = repo.FindCustomerByCPF(2, "12345678909")
	require.NoError(t, err)
	assert.Nil(t, byCPF)

	bySource, err := repo.FindCustomerBySource(1, models.SourceOnMotor, "77")
	require.NoError(t, err)
	assert.Equal(t, maria.ID, bySource.ID)
	bySource, err = repo.FindCustomerBySource(1, models.SourceOnMotor, "78")
	require.NoError(t, err)
	assert.Nil(t, bySource)

	maria.Name = "Maria Souza"
	require.NoError(t, repo.SaveCustomer(maria))
	got, _ = repo.FindCustomer(1, maria.ID)
	assert.Equal(t, "Maria Souza", got.Name)
}

func TestCustomerRepository_TouchCustomerSyncedKeepsUpdatedAt(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomerRepository(db)
	c := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "Maria"})
	old := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	require.NoError(t, db.Model(c).UpdateColumn("updated_at", old).Error)

	at := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	require.NoError(t, repo.TouchCustomerSynced(c.ID, at))
	got, _ := repo.FindCustomer(1, c.ID)
	require.NotNil(t, got.SyncedAt)
	assert.True(t, at.Equal(*got.SyncedAt))
	assert.True(t, old.Equal(got.UpdatedAt))
}

func TestCustomerRepository_SearchCustomers(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomerRepository(db)
	maria := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "Maria Souza", CPF: "12345678909", Email: "maria@exemplo.com"})
	joao := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "João Lima", Mobile: "11987654321", CNPJ: "11222333000181"})
	ana := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "Ana", Phone: "1133334444"})
	seedCustomer(t, repo, models.Customer{WorkspaceID: 2, Name: "Maria de Outra Oficina", CPF: "12345678909"})
	require.NoError(t, db.Create(&models.CustomerVehicle{WorkspaceID: 1, CustomerID: &ana.ID, Plate: "ABC1D23"}).Error)
	require.NoError(t, db.Create(&models.CustomerVehicle{WorkspaceID: 2, CustomerID: &joao.ID, Plate: "ZZZ9Z99"}).Error)

	search := func(term string) []uint {
		t.Helper()
		got, total, err := repo.SearchCustomers(1, CustomerFilter{Search: term, Page: 1, Limit: 20})
		require.NoError(t, err)
		ids := make([]uint, len(got))
		for i, c := range got {
			ids[i] = c.ID
		}
		assert.Equal(t, int64(len(got)), total)
		return ids
	}

	assert.Equal(t, []uint{maria.ID}, search("MARIA"), "nome sem diferenciar maiúsculas, só na oficina")
	assert.Equal(t, []uint{maria.ID}, search("Exemplo.COM"))
	assert.Equal(t, []uint{maria.ID}, search("123.456.789-09"), "CPF com máscara")
	assert.Equal(t, []uint{joao.ID}, search("(11) 98765"))
	assert.Equal(t, []uint{joao.ID}, search("11.222.333/0001"))
	assert.Equal(t, []uint{ana.ID}, search("3333-4444"))
	assert.Equal(t, []uint{ana.ID}, search("abc-1d23"), "placa do veículo do cliente")
	assert.Empty(t, search("zzz9z99"), "placa de outra oficina não casa")
	assert.Len(t, search(""), 3)

	// Mais recentemente atualizado primeiro; paginação.
	require.NoError(t, db.Model(&models.Customer{}).Where("id = ?", ana.ID).UpdateColumn("updated_at", time.Now().Add(-time.Hour)).Error)
	page1, total, err := repo.SearchCustomers(1, CustomerFilter{Page: 1, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, page1, 2)
	assert.Equal(t, joao.ID, page1[0].ID)
	page2, _, err := repo.SearchCustomers(1, CustomerFilter{Page: 2, Limit: 2})
	require.NoError(t, err)
	require.Len(t, page2, 1)
	assert.Equal(t, ana.ID, page2[0].ID)
}

func TestCustomerRepository_LastServiceAtAndVehicles(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomerRepository(db)
	maria := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "Maria"})
	joao := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "João"})
	ana := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "Ana"})

	osDate := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	older := time.Date(2025, 3, 1, 9, 0, 0, 0, time.UTC)
	ws := uint(1)
	require.NoError(t, db.Create(&models.ServiceRecord{WorkspaceID: 1, SourceID: "1", CustomerID: &maria.ID, OpenedAt: &older}).Error)
	require.NoError(t, db.Create(&models.ServiceRecord{WorkspaceID: 1, SourceID: "2", CustomerID: &maria.ID, OpenedAt: &osDate}).Error)
	require.NoError(t, db.Create(&models.ServiceRecord{WorkspaceID: 1, SourceID: "3", CustomerID: &joao.ID}).Error, "OS sem data não conta")

	list := &models.TaskList{Title: "Gol", UserID: 1, WorkspaceID: &ws, CustomerID: &joao.ID}
	require.NoError(t, db.Create(list).Error)
	deleted := &models.TaskList{Title: "Uno", UserID: 1, WorkspaceID: &ws, CustomerID: &ana.ID}
	require.NoError(t, db.Create(deleted).Error)
	require.NoError(t, db.Delete(deleted).Error)

	last, err := repo.LastServiceAt(1, []uint{maria.ID, joao.ID, ana.ID})
	require.NoError(t, err)
	assert.True(t, osDate.Equal(last[maria.ID]))
	assert.WithinDuration(t, time.Now(), last[joao.ID], time.Minute)
	assert.NotContains(t, last, ana.ID, "atendimento excluído não conta")

	empty, err := repo.LastServiceAt(1, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)

	require.NoError(t, db.Create(&models.CustomerVehicle{WorkspaceID: 1, CustomerID: &maria.ID, Plate: "BBB2B22"}).Error)
	require.NoError(t, db.Create(&models.CustomerVehicle{WorkspaceID: 1, CustomerID: &maria.ID, Plate: "AAA1A11"}).Error)
	require.NoError(t, db.Create(&models.CustomerVehicle{WorkspaceID: 1, Plate: "CCC3C33"}).Error)
	vehicles, err := repo.VehiclesByCustomers(1, []uint{maria.ID})
	require.NoError(t, err)
	require.Len(t, vehicles, 2)
	assert.Equal(t, "AAA1A11", vehicles[0].Plate)
	none, err := repo.VehiclesByCustomers(1, nil)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestScannedTime(t *testing.T) {
	ref := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	for _, in := range []any{ref, "2026-03-01 09:00:00+00:00", []byte("2026-03-01T09:00:00Z"), "2026-03-01 09:00:00"} {
		var s scannedTime
		require.NoError(t, s.Scan(in))
		assert.True(t, s.Valid)
		assert.True(t, ref.Equal(s.Time), in)
	}
	var s scannedTime
	require.NoError(t, s.Scan(nil))
	assert.False(t, s.Valid)
	assert.Error(t, s.Scan(12))
	assert.Error(t, s.Scan("ontem"))
}

func TestCustomerRepository_ErrorsWhenDatabaseIsClosed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomerRepository(db)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	_, err = repo.FindCustomerByCPF(1, "1")
	assert.Error(t, err)
	_, _, err = repo.SearchCustomers(1, CustomerFilter{Page: 1, Limit: 1})
	assert.Error(t, err)
	_, err = repo.LastServiceAt(1, []uint{1})
	assert.Error(t, err)
}

func TestCustomerRepository_SearchOrdersByLastVisit(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomerRepository(db)
	jan := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	nov := time.Date(2026, 11, 7, 17, 32, 0, 0, time.UTC)
	older := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "Visitou em janeiro", LastVisitAt: &jan})
	never := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "Nunca visitou"})
	recent := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "Visitou em novembro", LastVisitAt: &nov})
	neverOld := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "Nunca visitou, antigo"})
	require.NoError(t, db.Model(neverOld).UpdateColumn("updated_at", time.Now().Add(-48*time.Hour)).Error)
	// Atualizado agora, mas visitou há mais tempo: a visita manda na ordem.
	require.NoError(t, db.Model(older).UpdateColumn("updated_at", time.Now().Add(time.Hour)).Error)

	got, _, err := repo.SearchCustomers(1, CustomerFilter{Page: 1, Limit: 10})
	require.NoError(t, err)
	ids := make([]uint, len(got))
	for i, c := range got {
		ids[i] = c.ID
	}
	assert.Equal(t, []uint{recent.ID, older.ID, never.ID, neverOld.ID}, ids)
	require.NotNil(t, got[0].LastVisitAt)
	assert.True(t, nov.Equal(*got[0].LastVisitAt))
}
