package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func TestCustomerRepository_CustomersForImport(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomerRepository(db)
	bySource := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "A", Source: models.SourceOnMotor, SourceID: strPtr("cpf:1")})
	byCPF := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "B", CPF: "12345678909"})
	byCNPJ := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "C", CNPJ: "11222333000181"})
	byMobile := seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "D", Source: models.SourceOnMotor, SourceID: strPtr("nome:d|cel:1"), Mobile: "11987654321"})
	seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "Local sem CPF", Mobile: "11987654321"})
	seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "Com CPF", Source: models.SourceOnMotor, SourceID: strPtr("x"), CPF: "52998224725", Mobile: "11987654321"})
	seedCustomer(t, repo, models.Customer{WorkspaceID: 1, Name: "Sem CPF, fora do lote"})
	seedCustomer(t, repo, models.Customer{WorkspaceID: 2, Name: "Outra oficina", CPF: "12345678909"})

	got, err := repo.CustomersForImport(1, CustomerImportLookup{
		SourceIDs: []string{"cpf:1"}, CPFs: []string{"12345678909"}, CNPJs: []string{"11222333000181"}, Mobiles: []string{"11987654321"},
	})
	require.NoError(t, err)
	ids := []uint{}
	for _, c := range got {
		ids = append(ids, c.ID)
	}
	assert.Equal(t, []uint{bySource.ID, byCPF.ID, byCNPJ.ID, byMobile.ID}, ids,
		"celular só casa cliente do OnMotor sem CPF/CNPJ; listas vazias não casam clientes sem documento")

	none, err := repo.CustomersForImport(1, CustomerImportLookup{})
	require.NoError(t, err)
	assert.Empty(t, none)
	only, err := repo.CustomersForImport(1, CustomerImportLookup{CPFs: []string{"12345678909"}})
	require.NoError(t, err)
	require.Len(t, only, 1)
	assert.Equal(t, byCPF.ID, only[0].ID)
}

func TestCustomerRepository_VehiclesAndServicesForImport(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomerRepository(db)
	a := &models.CustomerVehicle{WorkspaceID: 1, Source: models.SourceOnMotor, SourceID: strPtr("placa:AAA1A11"), Plate: "AAA1A11"}
	b := &models.CustomerVehicle{WorkspaceID: 1, Plate: "BBB2B22"}
	c := &models.CustomerVehicle{WorkspaceID: 1}
	for _, v := range []*models.CustomerVehicle{a, b, c, {WorkspaceID: 2, Plate: "BBB2B22"}} {
		require.NoError(t, repo.CreateVehicle(v))
	}
	got, err := repo.VehiclesForImport(1, []string{"placa:AAA1A11"}, []string{"BBB2B22"})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, a.ID, got[0].ID)
	assert.Equal(t, b.ID, got[1].ID)
	none, err := repo.VehiclesForImport(1, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, none, "lista vazia não traz veículos sem placa")

	require.NoError(t, repo.CreateServiceRecords([]*models.ServiceRecord{{WorkspaceID: 1, Source: models.SourceOnMotor, SourceID: "os-1", Items: "[]"}}))
	records, err := repo.ServiceRecordsForImport(1, []string{"os-1", "os-2"})
	require.NoError(t, err)
	require.Len(t, records, 1)
	records, err = repo.ServiceRecordsForImport(1, nil)
	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestCustomerRepository_BulkWritesInTransaction(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomerRepository(db)
	at := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)

	created := []*models.Customer{
		{WorkspaceID: 1, Name: "A", CPF: "12345678909", Source: models.SourceOnMotor, SourceID: strPtr("cpf:12345678909")},
		{WorkspaceID: 1, Name: "B", Source: models.SourceOnMotor, SourceID: strPtr("nome:b|cel:1")},
	}
	require.NoError(t, repo.Transaction(func(tx *CustomerRepository) error { return tx.CreateCustomers(created) }))
	require.NotZero(t, created[0].ID)
	require.NotZero(t, created[1].ID)
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, db.Model(&models.Customer{}).Where("id = ?", created[0].ID).UpdateColumns(map[string]any{"created_at": old, "updated_at": old}).Error)

	// Troca de chaves entre dois clientes no mesmo lote: soltar antes, regravar depois.
	a, _ := repo.FindCustomer(1, created[0].ID)
	b, _ := repo.FindCustomer(1, created[1].ID)
	a.CPF, b.CPF = "", "12345678909"
	a.SourceID, b.SourceID = strPtr("nome:a|cel:2"), strPtr("cpf:12345678909")
	a.UpdatedAt, b.UpdatedAt = at, at
	require.NoError(t, repo.Transaction(func(tx *CustomerRepository) error {
		if err := tx.ReleaseCustomerKeys([]uint{a.ID, b.ID}); err != nil {
			return err
		}
		return tx.UpsertCustomers([]*models.Customer{a, b})
	}))
	gotA, _ := repo.FindCustomer(1, a.ID)
	gotB, _ := repo.FindCustomer(1, b.ID)
	assert.Equal(t, "", gotA.CPF)
	assert.Equal(t, "nome:a|cel:2", *gotA.SourceID)
	assert.Equal(t, "12345678909", gotB.CPF)
	assert.True(t, old.Equal(gotA.CreatedAt), "upsert preserva created_at")
	assert.True(t, gotA.UpdatedAt.After(old), "upsert renova updated_at")
	upserted := gotA.UpdatedAt

	require.NoError(t, repo.TouchCustomers([]uint{a.ID}, at.Add(time.Hour)))
	gotA, _ = repo.FindCustomer(1, a.ID)
	assert.True(t, at.Add(time.Hour).Equal(*gotA.SyncedAt))
	assert.True(t, upserted.Equal(gotA.UpdatedAt), "touch não mexe em updated_at")

	// Erro dentro da transação desfaz tudo.
	err := repo.Transaction(func(tx *CustomerRepository) error {
		if err := tx.CreateCustomers([]*models.Customer{{WorkspaceID: 1, Name: "Z"}}); err != nil {
			return err
		}
		return assert.AnError
	})
	assert.ErrorIs(t, err, assert.AnError)
	var count int64
	db.Model(&models.Customer{}).Count(&count)
	assert.Equal(t, int64(2), count)

	vehicles := []*models.CustomerVehicle{{WorkspaceID: 1, Plate: "AAA1A11", Source: models.SourceOnMotor, SourceID: strPtr("placa:AAA1A11")}, {WorkspaceID: 1, Plate: "BBB2B22"}}
	require.NoError(t, repo.CreateVehicles(vehicles))
	vehicles[0].Plate, vehicles[1].Plate = "BBB2B22", "AAA1A11"
	require.NoError(t, repo.ReleaseVehicleKeys([]uint{vehicles[0].ID, vehicles[1].ID}))
	require.NoError(t, repo.UpsertVehicles(vehicles))
	require.NoError(t, repo.TouchVehicles([]uint{vehicles[0].ID}, at))
	v0, _ := repo.FindVehicle(1, vehicles[0].ID)
	assert.Equal(t, "BBB2B22", v0.Plate)
	assert.True(t, at.Equal(*v0.SyncedAt))

	records := []*models.ServiceRecord{{WorkspaceID: 1, Source: models.SourceOnMotor, SourceID: "os-9", Items: "[]"}}
	require.NoError(t, repo.CreateServiceRecords(records))
	records[0].Status = "Entregue"
	require.NoError(t, repo.UpsertServiceRecords(records))
	require.NoError(t, repo.TouchServiceRecords([]uint{records[0].ID}, at))
	got, _ := repo.ServiceRecordsForImport(1, []string{"os-9"})
	assert.Equal(t, "Entregue", got[0].Status)
	assert.True(t, at.Equal(*got[0].SyncedAt))

	for _, err := range []error{
		repo.CreateCustomers(nil), repo.UpsertCustomers(nil), repo.ReleaseCustomerKeys(nil), repo.TouchCustomers(nil, at),
	} {
		assert.NoError(t, err, "listas vazias não vão ao banco")
	}
}
