package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func (f *customerFixture) customer(t *testing.T, name string) *CustomerDetail {
	t.Helper()
	c, err := f.svc.CreateCustomer(fxOwner, CustomerInput{Name: ptr(name)})
	require.NoError(t, err)
	return c
}

func TestCustomerService_CreateVehicle(t *testing.T) {
	f := newCustomerFixture(t)
	maria := f.customer(t, "Maria")

	v, err := f.svc.CreateVehicle(fxManager, maria.ID, CustomerVehicleInput{
		Plate: ptr(" abc-1d23 "), Manufacturer: ptr("VW"), Model: ptr(" Gol 1.6 "),
		YearFab: ptr(2014), YearModel: ptr(2015), Doors: ptr(4), LastMileage: ptr(87500),
		CustomerID: ptr(uint(999)),
	})
	require.NoError(t, err)
	assert.Equal(t, "ABC1D23", v.Plate)
	assert.Equal(t, "Gol 1.6", v.Model)
	assert.Equal(t, models.SourceLocal, v.Source)
	assert.Equal(t, maria.ID, *v.CustomerID, "o dono vem da rota, não do body")
	assert.Equal(t, 2015, *v.YearModel)
	assert.Equal(t, 87500, *v.LastMileage)

	list, err := f.svc.ListVehicles(fxOwner, maria.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)

	// Placa repetida na oficina → 409 com o id existente (com ou sem máscara).
	joao := f.customer(t, "João")
	_, err = f.svc.CreateVehicle(fxOwner, joao.ID, CustomerVehicleInput{Plate: ptr("ABC1D23")})
	var dup *DuplicateError
	require.ErrorAs(t, err, &dup)
	assert.ErrorIs(t, err, ErrPlateTaken)
	assert.Equal(t, v.ID, dup.ExistingID)
}

func TestCustomerService_CreateVehicle_Validation(t *testing.T) {
	f := newCustomerFixture(t)
	maria := f.customer(t, "Maria")
	cases := []struct {
		in   CustomerVehicleInput
		want error
	}{
		{CustomerVehicleInput{}, ErrPlateRequired},
		{CustomerVehicleInput{Plate: ptr(" - ")}, ErrPlateRequired},
		{CustomerVehicleInput{Plate: ptr("ABC1234"), YearFab: ptr(1800)}, ErrInvalidYear},
		{CustomerVehicleInput{Plate: ptr("ABC1234"), YearModel: ptr(3000)}, ErrInvalidYear},
		{CustomerVehicleInput{Plate: ptr("ABC1234"), Doors: ptr(-1)}, ErrInvalidDoors},
		{CustomerVehicleInput{Plate: ptr("ABC1234"), LastMileage: ptr(-5)}, ErrInvalidMileage},
	}
	for _, tc := range cases {
		_, err := f.svc.CreateVehicle(fxOwner, maria.ID, tc.in)
		assert.ErrorIs(t, err, tc.want)
	}
	_, err := f.svc.CreateVehicle(fxOwner, 999, CustomerVehicleInput{Plate: ptr("ABC1234")})
	assert.ErrorIs(t, err, ErrCustomerNotFound)
	_, err = f.svc.ListVehicles(fxOwner, 999)
	assert.ErrorIs(t, err, ErrCustomerNotFound)
}

func TestCustomerService_UpdateVehicle(t *testing.T) {
	f := newCustomerFixture(t)
	maria := f.customer(t, "Maria")
	joao := f.customer(t, "João")
	v, err := f.svc.CreateVehicle(fxOwner, maria.ID, CustomerVehicleInput{Plate: ptr("ABC1D23"), Model: ptr("Gol"), Color: ptr("Prata")})
	require.NoError(t, err)
	other, err := f.svc.CreateVehicle(fxOwner, joao.ID, CustomerVehicleInput{Plate: ptr("XYZ9W87")})
	require.NoError(t, err)

	updated, err := f.svc.UpdateVehicle(fxManager, v.ID, CustomerVehicleInput{Color: ptr("Preto"), CustomerID: &joao.ID, LastMileage: ptr(90000)})
	require.NoError(t, err)
	assert.Equal(t, "Preto", updated.Color)
	assert.Equal(t, "Gol", updated.Model, "ausente mantém")
	assert.Equal(t, "ABC1D23", updated.Plate)
	assert.Equal(t, joao.ID, *updated.CustomerID, "transferido para outro cliente")

	_, err = f.svc.UpdateVehicle(fxOwner, v.ID, CustomerVehicleInput{Plate: ptr("xyz-9w87")})
	var dup *DuplicateError
	require.ErrorAs(t, err, &dup)
	assert.Equal(t, other.ID, dup.ExistingID)

	_, err = f.svc.UpdateVehicle(fxOwner, v.ID, CustomerVehicleInput{Plate: ptr("")})
	assert.ErrorIs(t, err, ErrPlateRequired)
	_, err = f.svc.UpdateVehicle(fxOwner, v.ID, CustomerVehicleInput{YearFab: ptr(1)})
	assert.ErrorIs(t, err, ErrInvalidYear)

	foreign := &models.Customer{WorkspaceID: fxOtherWorkspc, Name: "Outra"}
	require.NoError(t, f.db.Create(foreign).Error)
	_, err = f.svc.UpdateVehicle(fxOwner, v.ID, CustomerVehicleInput{CustomerID: &foreign.ID})
	assert.ErrorIs(t, err, ErrInvalidCustomer)

	foreignVehicle := &models.CustomerVehicle{WorkspaceID: fxOtherWorkspc, Plate: "OUT1R00"}
	require.NoError(t, f.db.Create(foreignVehicle).Error)
	_, err = f.svc.UpdateVehicle(fxOwner, foreignVehicle.ID, CustomerVehicleInput{Color: ptr("x")})
	assert.ErrorIs(t, err, ErrCustomerVehicleNotFound)

	// Veículo importado sem placa pode ser editado sem informar placa.
	noPlate := &models.CustomerVehicle{WorkspaceID: fxWorkspaceID, Source: models.SourceOnMotor, SourceID: ptr("9")}
	require.NoError(t, f.db.Create(noPlate).Error)
	require.NoError(t, f.db.Create(&models.CustomerVehicle{WorkspaceID: fxWorkspaceID}).Error)
	_, err = f.svc.UpdateVehicle(fxOwner, noPlate.ID, CustomerVehicleInput{Color: ptr("Azul")})
	require.NoError(t, err)
}

func TestCustomerService_LookupByPlate(t *testing.T) {
	f := newCustomerFixture(t)
	maria := f.customer(t, "Maria")
	v, err := f.svc.CreateVehicle(fxOwner, maria.ID, CustomerVehicleInput{Plate: ptr("ABC1D23")})
	require.NoError(t, err)
	orphan := &models.CustomerVehicle{WorkspaceID: fxWorkspaceID, Plate: "SEM1D00"}
	require.NoError(t, f.db.Create(orphan).Error)
	require.NoError(t, f.db.Create(&models.CustomerVehicle{WorkspaceID: fxOtherWorkspc, Plate: "OUT1R00"}).Error)

	found, err := f.svc.LookupByPlate(fxManager, "abc-1d23")
	require.NoError(t, err)
	assert.Equal(t, v.ID, found.Vehicle.ID)
	require.NotNil(t, found.Customer)
	assert.Equal(t, "Maria", found.Customer.Name)

	found, err = f.svc.LookupByPlate(fxOwner, "sem1d00")
	require.NoError(t, err)
	assert.Nil(t, found.Customer)
	assert.Contains(t, jsonKeys(t, found), "customer", "customer vem como null")

	_, err = f.svc.LookupByPlate(fxOwner, "OUT1R00")
	assert.ErrorIs(t, err, ErrCustomerVehicleNotFound, "placa de outra oficina → 404")
	_, err = f.svc.LookupByPlate(fxOwner, "ZZZ0Z00")
	assert.ErrorIs(t, err, ErrCustomerVehicleNotFound)
	_, err = f.svc.LookupByPlate(fxOwner, " ")
	assert.ErrorIs(t, err, ErrPlateRequired)
}

func TestCustomerService_VehicleEndpointsPermissions(t *testing.T) {
	f := newCustomerFixture(t)
	maria := f.customer(t, "Maria")
	v, err := f.svc.CreateVehicle(fxOwner, maria.ID, CustomerVehicleInput{Plate: ptr("ABC1D23")})
	require.NoError(t, err)

	_, err = f.svc.ListVehicles(fxEditor, maria.ID)
	assert.ErrorIs(t, err, ErrCustomerForbidden)
	_, err = f.svc.CreateVehicle(fxEditor, maria.ID, CustomerVehicleInput{Plate: ptr("X")})
	assert.ErrorIs(t, err, ErrCustomerForbidden)
	_, err = f.svc.UpdateVehicle(fxEditor, v.ID, CustomerVehicleInput{})
	assert.ErrorIs(t, err, ErrCustomerForbidden)
	_, err = f.svc.LookupByPlate(fxEditor, "ABC1D23")
	assert.ErrorIs(t, err, ErrCustomerForbidden)
	_, err = f.svc.VehicleHistory(fxEditor, v.ID)
	assert.ErrorIs(t, err, ErrCustomerForbidden)
	_, err = f.svc.LookupByPlate(fxOutsider, "ABC1D23")
	assert.ErrorIs(t, err, ErrNoWorkspace)
}

func TestCustomerService_VehicleHistory_MergesNewestFirst(t *testing.T) {
	f := newCustomerFixture(t)
	maria := f.customer(t, "Maria")
	v, err := f.svc.CreateVehicle(fxOwner, maria.ID, CustomerVehicleInput{Plate: ptr("ABC1D23")})
	require.NoError(t, err)

	brt := time.FixedZone("BRT", -3*3600)
	jan := time.Date(2026, 1, 10, 9, 0, 0, 0, brt)
	mar := time.Date(2026, 3, 10, 9, 0, 0, 0, brt)
	delivered := time.Date(2026, 3, 12, 17, 0, 0, 0, brt)
	total := 450.5
	withItems := &models.ServiceRecord{WorkspaceID: fxWorkspaceID, SourceID: "100", CustomerVehicleID: &v.ID, OpenedAt: &mar, DeliveredAt: &delivered,
		Mileage: ptr(80000), Status: "Entregue", Type: "Revisão", Total: &total,
		Items: `[{"kind":"servico","description":"Troca de óleo","quantity":1,"unit_price":100,"total":100},{"kind":"peca","description":"Filtro"},{"kind":"peca","description":" "},{"kind":"peca","description":"Vela"},{"kind":"peca","description":"Correia"}]`}
	withNotes := &models.ServiceRecord{WorkspaceID: fxWorkspaceID, SourceID: "90", CustomerVehicleID: &v.ID, OpenedAt: &jan, Notes: "Barulho na suspensão", Items: "[]"}
	undated := &models.ServiceRecord{WorkspaceID: fxWorkspaceID, SourceID: "80", CustomerVehicleID: &v.ID, Items: "inválido"}
	require.NoError(t, f.db.Create(withItems).Error)
	require.NoError(t, f.db.Create(withNotes).Error)
	require.NoError(t, f.db.Create(undated).Error)

	ws := fxWorkspaceID
	ours := &models.TaskList{Title: "Gol 2015", UserID: fxOwner, WorkspaceID: &ws, CustomerVehicleID: &v.ID, Status: models.StatusEmAndamento, Mileage: ptr(91000)}
	require.NoError(t, f.db.Create(ours).Error)
	require.NoError(t, f.db.Create(&models.TaskItem{TaskListID: ours.ID, Text: "Alinhamento"}).Error)
	bare := &models.TaskList{Title: "Só título", UserID: fxOwner, WorkspaceID: &ws, CustomerVehicleID: &v.ID}
	require.NoError(t, f.db.Create(bare).Error)
	require.NoError(t, f.db.Model(bare).UpdateColumn("created_at", time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)).Error)

	history, err := f.svc.VehicleHistory(fxManager, v.ID)
	require.NoError(t, err)
	require.Len(t, history.Records, 5)
	r := history.Records

	assert.Equal(t, HistorySourceRMAXX, r[0].Source, "nosso atendimento de hoje é o mais recente")
	assert.Equal(t, ours.ID, *r[0].ListID)
	assert.Equal(t, "Alinhamento", r[0].Summary)
	assert.Equal(t, "em_andamento", r[0].Status)
	assert.Equal(t, 91000, *r[0].Mileage)

	assert.Equal(t, HistorySourceOnMotor, r[1].Source)
	assert.Equal(t, withItems.ID, r[1].ID)
	assert.Nil(t, r[1].ListID)
	assert.Equal(t, "Troca de óleo; Filtro; Vela", r[1].Summary, "sem notas: até 3 itens")
	assert.Equal(t, "Revisão", r[1].Type)
	assert.Equal(t, 450.5, *r[1].Total)
	assert.True(t, delivered.Equal(*r[1].DeliveredAt))
	assert.Equal(t, time.UTC, r[1].OpenedAt.Location())

	assert.Equal(t, "Só título", r[2].Summary, "atendimento sem serviços usa o título")
	assert.Equal(t, "Barulho na suspensão", r[3].Summary)
	assert.Equal(t, withNotes.ID, r[3].ID)
	assert.Nil(t, r[4].OpenedAt, "OS sem data vai para o fim")

	body := jsonKeys(t, history)
	records := body["records"].([]any)
	assert.Nil(t, records[0].(map[string]any)["items"], "atendimento nosso: items null")
	assert.Len(t, records[1].(map[string]any)["items"], 5)
	assert.Equal(t, []any{}, records[4].(map[string]any)["items"], "items inválido vira []")

	_, err = f.svc.VehicleHistory(fxOwner, 999)
	assert.ErrorIs(t, err, ErrCustomerVehicleNotFound)
}

func TestHistoryHelpers(t *testing.T) {
	assert.Equal(t, "abc", truncateRunes("abc", 5))
	assert.Equal(t, "ab…", truncateRunes("abcdef", 3))
	assert.Nil(t, utcPtr(nil))

	a := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	records := []HistoryRecord{{ID: 1}, {ID: 2, OpenedAt: &a}, {ID: 3}, {ID: 4, OpenedAt: &a}}
	sortHistory(records)
	ids := []uint{records[0].ID, records[1].ID, records[2].ID, records[3].ID}
	assert.Equal(t, []uint{4, 2, 3, 1}, ids)
}

func TestCustomerService_VehicleStoreErrors(t *testing.T) {
	f := newCustomerFixture(t)
	maria := f.customer(t, "Maria")
	v, err := f.svc.CreateVehicle(fxOwner, maria.ID, CustomerVehicleInput{Plate: ptr("ABC1D23")})
	require.NoError(t, err)

	boom := assert.AnError
	svc := NewCustomerService(&vehicleErrStore{CustomerStore: f.repo, plateErr: boom}, fixtureWorkspaceStore())
	_, err = svc.CreateVehicle(fxOwner, maria.ID, CustomerVehicleInput{Plate: ptr("NEW1A11")})
	assert.ErrorIs(t, err, boom)
	_, err = svc.LookupByPlate(fxOwner, "ABC1D23")
	assert.ErrorIs(t, err, boom)

	svc = NewCustomerService(&vehicleErrStore{CustomerStore: f.repo, writeErr: boom}, fixtureWorkspaceStore())
	_, err = svc.CreateVehicle(fxOwner, maria.ID, CustomerVehicleInput{Plate: ptr("NEW1A11")})
	assert.ErrorIs(t, err, boom)
	_, err = svc.UpdateVehicle(fxOwner, v.ID, CustomerVehicleInput{Color: ptr("x")})
	assert.ErrorIs(t, err, boom)

	svc = NewCustomerService(&vehicleErrStore{CustomerStore: f.repo, recordsErr: boom}, fixtureWorkspaceStore())
	_, err = svc.VehicleHistory(fxOwner, v.ID)
	assert.ErrorIs(t, err, boom)
	svc = NewCustomerService(&vehicleErrStore{CustomerStore: f.repo, listsErr: boom}, fixtureWorkspaceStore())
	_, err = svc.VehicleHistory(fxOwner, v.ID)
	assert.ErrorIs(t, err, boom)
}

type vehicleErrStore struct {
	CustomerStore
	plateErr, writeErr, recordsErr, listsErr error
}

func (s *vehicleErrStore) FindVehicleByPlate(workspaceID uint, plate string) (*models.CustomerVehicle, error) {
	if s.plateErr != nil {
		return nil, s.plateErr
	}
	return s.CustomerStore.FindVehicleByPlate(workspaceID, plate)
}
func (s *vehicleErrStore) CreateVehicle(v *models.CustomerVehicle) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	return s.CustomerStore.CreateVehicle(v)
}
func (s *vehicleErrStore) SaveVehicle(v *models.CustomerVehicle) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	return s.CustomerStore.SaveVehicle(v)
}
func (s *vehicleErrStore) ServiceRecordsByVehicle(workspaceID, vehicleID uint) ([]models.ServiceRecord, error) {
	if s.recordsErr != nil {
		return nil, s.recordsErr
	}
	return s.CustomerStore.ServiceRecordsByVehicle(workspaceID, vehicleID)
}
func (s *vehicleErrStore) TaskListsByCustomerVehicle(workspaceID, vehicleID uint) ([]models.TaskList, error) {
	if s.listsErr != nil {
		return nil, s.listsErr
	}
	return s.CustomerStore.TaskListsByCustomerVehicle(workspaceID, vehicleID)
}
