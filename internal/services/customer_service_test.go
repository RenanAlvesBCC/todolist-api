package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
)

type customerFixture struct {
	db   *gorm.DB
	repo *repository.CustomerRepository
	svc  *CustomerService
}

func fixtureWorkspaceStore() *mockWsCtxStore {
	return &mockWsCtxStore{
		workspace: &models.Workspace{Base: models.Base{ID: fxWorkspaceID}},
		memberRole: map[uint]models.WorkspaceRole{
			fxOwner:       models.RoleOwner,
			fxManager:     models.RoleManager,
			fxEditor:      models.RoleEditor,
			fxEditorOther: models.RoleEditor,
		},
	}
}

func newCustomerFixture(t *testing.T) *customerFixture {
	t.Helper()
	db := setupTestDB(t)
	repo := repository.NewCustomerRepository(db)
	svc := NewCustomerService(repo, fixtureWorkspaceStore())
	svc.now = func() time.Time { return time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC) }
	return &customerFixture{db: db, repo: repo, svc: svc}
}

func fullCustomerInput() CustomerInput {
	return CustomerInput{
		Name:           ptr("  Maria Souza "),
		PersonType:     ptr("PF"),
		CPF:            ptr("123.456.789-09"),
		RG:             ptr(" 12.345.678-9 "),
		Phone:          ptr("(11) 3333-4444"),
		Mobile:         ptr("+55 (11) 98765-4321"),
		Email:          ptr("Maria@Exemplo.com"),
		EmailSecondary: ptr(""),
		CEP:            ptr("01310-100"),
		Street:         ptr("Av. Paulista"),
		Number:         ptr("1000"),
		Complement:     ptr("ap 1"),
		District:       ptr("Bela Vista"),
		City:           ptr("São Paulo"),
		UF:             ptr("sp"),
		BirthDate:      ptr("03/05/1980"),
		Notes:          ptr("prefere WhatsApp"),
	}
}

func TestCustomerService_CreateCustomer_NormalizesAndStores(t *testing.T) {
	f := newCustomerFixture(t)
	created, err := f.svc.CreateCustomer(fxManager, fullCustomerInput())
	require.NoError(t, err)

	assert.Equal(t, models.SourceLocal, created.Source)
	assert.Nil(t, created.SourceID)
	assert.Equal(t, fxWorkspaceID, created.WorkspaceID)
	assert.Empty(t, created.Vehicles)

	got, err := f.repo.FindCustomer(fxWorkspaceID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Maria Souza", got.Name)
	assert.Equal(t, "pf", got.PersonType)
	assert.Equal(t, "12345678909", got.CPF)
	assert.Equal(t, "12.345.678-9", got.RG)
	assert.Equal(t, "1133334444", got.Phone)
	assert.Equal(t, "11987654321", got.Mobile)
	assert.Equal(t, "maria@exemplo.com", got.Email)
	assert.Equal(t, "01310100", got.CEP)
	assert.Equal(t, "SP", got.UF)
	require.NotNil(t, got.BirthDate)
	assert.Equal(t, "1980-05-03", got.BirthDate.String())
	assert.Equal(t, "prefere WhatsApp", got.Notes)
}

func TestCustomerService_CreateCustomer_Validation(t *testing.T) {
	f := newCustomerFixture(t)
	cases := []struct {
		in   CustomerInput
		want error
	}{
		{CustomerInput{}, ErrCustomerNameRequired},
		{CustomerInput{Name: ptr("  ")}, ErrCustomerNameRequired},
		{CustomerInput{Name: ptr("Maria"), CPF: ptr("111.111.111-11")}, ErrInvalidCPF},
		{CustomerInput{Name: ptr("Maria"), CNPJ: ptr("11.222.333/0001-80")}, ErrInvalidCNPJ},
		{CustomerInput{Name: ptr("Maria"), CEP: ptr("123")}, ErrInvalidCEP},
		{CustomerInput{Name: ptr("Maria"), Phone: ptr("123")}, ErrInvalidPhone},
		{CustomerInput{Name: ptr("Maria"), Mobile: ptr("123")}, ErrInvalidMobile},
		{CustomerInput{Name: ptr("Maria"), Email: ptr("maria")}, ErrInvalidEmail},
		{CustomerInput{Name: ptr("Maria"), PersonType: ptr("x")}, ErrInvalidPersonType},
		{CustomerInput{Name: ptr("Maria"), UF: ptr("XYZ")}, ErrInvalidUF},
		{CustomerInput{Name: ptr("Maria"), BirthDate: ptr("31/02/1980")}, ErrInvalidBirthDate},
	}
	for _, tc := range cases {
		_, err := f.svc.CreateCustomer(fxOwner, tc.in)
		assert.ErrorIs(t, err, tc.want)
	}
	var count int64
	f.db.Model(&models.Customer{}).Count(&count)
	assert.Zero(t, count, "nada é gravado com dado inválido")
}

func TestCustomerService_CreateCustomer_DuplicateCPFReturnsExistingID(t *testing.T) {
	f := newCustomerFixture(t)
	first, err := f.svc.CreateCustomer(fxOwner, CustomerInput{Name: ptr("Maria"), CPF: ptr("12345678909")})
	require.NoError(t, err)

	_, err = f.svc.CreateCustomer(fxManager, CustomerInput{Name: ptr("Outra"), CPF: ptr("123.456.789-09")})
	var dup *DuplicateError
	require.ErrorAs(t, err, &dup)
	assert.ErrorIs(t, err, ErrCustomerCPFTaken)
	assert.Equal(t, first.ID, dup.ExistingID)

	// Mesmo CPF em outra oficina não conflita.
	require.NoError(t, f.db.Create(&models.Customer{WorkspaceID: fxOtherWorkspc, Name: "X", CPF: "52998224725"}).Error)
	_, err = f.svc.CreateCustomer(fxOwner, CustomerInput{Name: ptr("Y"), CPF: ptr("529.982.247-25")})
	require.NoError(t, err)
}

func TestCustomerService_Permissions(t *testing.T) {
	f := newCustomerFixture(t)
	created, err := f.svc.CreateCustomer(fxOwner, CustomerInput{Name: ptr("Maria")})
	require.NoError(t, err)
	other := &models.Customer{WorkspaceID: fxOtherWorkspc, Name: "Outra oficina"}
	require.NoError(t, f.db.Create(other).Error)

	for _, user := range []uint{fxEditor, fxEditorOther} {
		_, err = f.svc.ListCustomers(user, "", 1, 20)
		assert.ErrorIs(t, err, ErrCustomerForbidden)
		_, err = f.svc.GetCustomer(user, created.ID)
		assert.ErrorIs(t, err, ErrCustomerForbidden)
		_, err = f.svc.CreateCustomer(user, CustomerInput{Name: ptr("x")})
		assert.ErrorIs(t, err, ErrCustomerForbidden)
		_, err = f.svc.UpdateCustomer(user, created.ID, CustomerInput{Name: ptr("x")})
		assert.ErrorIs(t, err, ErrCustomerForbidden)
	}

	_, err = f.svc.ListCustomers(fxOutsider, "", 1, 20)
	assert.ErrorIs(t, err, ErrNoWorkspace)

	for _, user := range []uint{fxOwner, fxManager} {
		_, err = f.svc.GetCustomer(user, created.ID)
		require.NoError(t, err)
		_, err = f.svc.GetCustomer(user, other.ID)
		assert.ErrorIs(t, err, ErrCustomerNotFound, "outra oficina → 404")
		_, err = f.svc.UpdateCustomer(user, other.ID, CustomerInput{Name: ptr("x")})
		assert.ErrorIs(t, err, ErrCustomerNotFound)
	}
}

func TestCustomerService_UpdateCustomer_AbsentKeepsPresentOverwrites(t *testing.T) {
	f := newCustomerFixture(t)
	created, err := f.svc.CreateCustomer(fxOwner, fullCustomerInput())
	require.NoError(t, err)

	updated, err := f.svc.UpdateCustomer(fxManager, created.ID, CustomerInput{City: ptr("Campinas"), Email: ptr(""), BirthDate: ptr("")})
	require.NoError(t, err)
	assert.Equal(t, "Campinas", updated.City)
	assert.Equal(t, "", updated.Email, "\"\" limpa")
	assert.Nil(t, updated.BirthDate)
	assert.Equal(t, "Maria Souza", updated.Name, "ausente mantém")
	assert.Equal(t, "12345678909", updated.CPF)
	assert.Equal(t, "11987654321", updated.Mobile)

	_, err = f.svc.UpdateCustomer(fxManager, created.ID, CustomerInput{Name: ptr("")})
	assert.ErrorIs(t, err, ErrCustomerNameRequired)
	_, err = f.svc.UpdateCustomer(fxManager, created.ID, CustomerInput{CPF: ptr("123")})
	assert.ErrorIs(t, err, ErrInvalidCPF)

	// Trocar para um CPF de outro cliente → 409 com o id dele; manter o próprio CPF é ok.
	joao, err := f.svc.CreateCustomer(fxOwner, CustomerInput{Name: ptr("João"), CPF: ptr("52998224725")})
	require.NoError(t, err)
	_, err = f.svc.UpdateCustomer(fxOwner, created.ID, CustomerInput{CPF: ptr("529.982.247-25")})
	var dup *DuplicateError
	require.ErrorAs(t, err, &dup)
	assert.Equal(t, joao.ID, dup.ExistingID)
	_, err = f.svc.UpdateCustomer(fxOwner, created.ID, CustomerInput{CPF: ptr("12345678909")})
	require.NoError(t, err)
}

func TestCustomerService_UpdateCustomer_OnMotorCustomerIsEditable(t *testing.T) {
	f := newCustomerFixture(t)
	c := &models.Customer{WorkspaceID: fxWorkspaceID, Name: "Importado", Source: models.SourceOnMotor, SourceID: ptr("55")}
	require.NoError(t, f.db.Create(c).Error)

	updated, err := f.svc.UpdateCustomer(fxOwner, c.ID, CustomerInput{Notes: ptr("cliente antigo")})
	require.NoError(t, err)
	assert.Equal(t, models.SourceOnMotor, updated.Source)
	assert.Equal(t, "55", *updated.SourceID)
	assert.Equal(t, "cliente antigo", updated.Notes)
}

func TestCustomerService_ListCustomers_WithVehiclesAndLastService(t *testing.T) {
	f := newCustomerFixture(t)
	maria, err := f.svc.CreateCustomer(fxOwner, CustomerInput{Name: ptr("Maria"), CPF: ptr("12345678909")})
	require.NoError(t, err)
	_, err = f.svc.CreateCustomer(fxOwner, CustomerInput{Name: ptr("João")})
	require.NoError(t, err)
	opened := time.Date(2026, 2, 1, 8, 0, 0, 0, time.FixedZone("BRT", -3*3600))
	require.NoError(t, f.db.Create(&models.CustomerVehicle{WorkspaceID: fxWorkspaceID, CustomerID: &maria.ID, Plate: "ABC1D23", Model: "Gol", YearModel: ptr(2015), LastMileage: ptr(87000), Chassis: "9BW"}).Error)
	require.NoError(t, f.db.Create(&models.ServiceRecord{WorkspaceID: fxWorkspaceID, SourceID: "1", CustomerID: &maria.ID, OpenedAt: &opened}).Error)

	page, err := f.svc.ListCustomers(fxManager, "abc1d23", 0, 500)
	require.NoError(t, err)
	assert.Equal(t, 1, page.Page)
	assert.Equal(t, 20, page.Limit, "limite fora da faixa volta para 20")
	assert.Equal(t, int64(1), page.Total)
	assert.Equal(t, 1, page.TotalPages)
	require.Len(t, page.Customers, 1)
	item := page.Customers[0]
	assert.Equal(t, "Maria", item.Name)
	require.Len(t, item.Vehicles, 1)
	assert.Equal(t, "ABC1D23", item.Vehicles[0].Plate)
	assert.Equal(t, 2015, *item.Vehicles[0].YearModel)
	require.NotNil(t, item.LastServiceAt)
	assert.True(t, opened.Equal(*item.LastServiceAt))
	assert.Equal(t, time.UTC, item.LastServiceAt.Location())

	body := jsonKeys(t, item)
	assert.Equal(t, "12345678909", body["cpf"])
	vehicle := body["vehicles"].([]any)[0].(map[string]any)
	assert.NotContains(t, vehicle, "chassis", "lista traz o veículo compacto")

	all, err := f.svc.ListCustomers(fxOwner, "", 1, 20)
	require.NoError(t, err)
	assert.Len(t, all.Customers, 2)
	for _, c := range all.Customers {
		if c.Name == "João" {
			assert.Nil(t, c.LastServiceAt)
			assert.NotNil(t, c.Vehicles)
			assert.Empty(t, c.Vehicles)
		}
	}

	detail, err := f.svc.GetCustomer(fxOwner, maria.ID)
	require.NoError(t, err)
	require.Len(t, detail.Vehicles, 1)
	assert.Equal(t, "9BW", detail.Vehicles[0].Chassis, "detalhe traz o veículo completo")
	require.NotNil(t, detail.LastServiceAt)
}

func TestCustomerService_StoreErrorsArePropagated(t *testing.T) {
	f := newCustomerFixture(t)
	created, err := f.svc.CreateCustomer(fxOwner, CustomerInput{Name: ptr("Maria")})
	require.NoError(t, err)
	sqlDB, err := f.db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	_, err = f.svc.ListCustomers(fxOwner, "", 1, 20)
	assert.Error(t, err)
	_, err = f.svc.CreateCustomer(fxOwner, CustomerInput{Name: ptr("x"), CPF: ptr("12345678909")})
	assert.Error(t, err)
	_, err = f.svc.CreateCustomer(fxOwner, CustomerInput{Name: ptr("x")})
	assert.Error(t, err)
	_, err = f.svc.UpdateCustomer(fxOwner, created.ID, CustomerInput{})
	assert.ErrorIs(t, err, ErrCustomerNotFound)
}

type failingCustomerStore struct {
	CustomerStore
	customer   *models.Customer
	saveErr    error
	vehicleErr error
	lastErr    error
}

func (s *failingCustomerStore) FindCustomer(workspaceID, id uint) (*models.Customer, error) {
	c := *s.customer
	return &c, nil
}
func (s *failingCustomerStore) FindCustomerByCPF(workspaceID uint, cpf string) (*models.Customer, error) {
	return nil, nil
}
func (s *failingCustomerStore) SaveCustomer(c *models.Customer) error { return s.saveErr }
func (s *failingCustomerStore) SearchCustomers(workspaceID uint, filter repository.CustomerFilter) ([]models.Customer, int64, error) {
	return []models.Customer{*s.customer}, 1, nil
}
func (s *failingCustomerStore) VehiclesByCustomers(workspaceID uint, ids []uint) ([]models.CustomerVehicle, error) {
	return nil, s.vehicleErr
}
func (s *failingCustomerStore) LastServiceAt(workspaceID uint, ids []uint) (map[uint]time.Time, error) {
	return nil, s.lastErr
}

func TestCustomerService_PartialStoreFailures(t *testing.T) {
	boom := assert.AnError
	customer := &models.Customer{ID: 1, WorkspaceID: fxWorkspaceID, Name: "Maria"}

	svc := NewCustomerService(&failingCustomerStore{customer: customer, saveErr: boom}, fixtureWorkspaceStore())
	_, err := svc.UpdateCustomer(fxOwner, 1, CustomerInput{})
	assert.ErrorIs(t, err, boom)

	svc = NewCustomerService(&failingCustomerStore{customer: customer, vehicleErr: boom}, fixtureWorkspaceStore())
	_, err = svc.ListCustomers(fxOwner, "", 1, 20)
	assert.ErrorIs(t, err, boom)
	_, err = svc.GetCustomer(fxOwner, 1)
	assert.ErrorIs(t, err, boom)

	svc = NewCustomerService(&failingCustomerStore{customer: customer, lastErr: boom}, fixtureWorkspaceStore())
	_, err = svc.GetCustomer(fxOwner, 1)
	assert.ErrorIs(t, err, boom)
}

func TestMemberWorkspace_RoleLookupFailure(t *testing.T) {
	store := &roleErrorStore{}
	_, _, err := memberWorkspace(store, 1)
	assert.ErrorIs(t, err, ErrNoWorkspace)
}

type roleErrorStore struct{}

func (roleErrorStore) FindByMemberUserID(userID uint) (*models.Workspace, error) {
	return &models.Workspace{Base: models.Base{ID: 1}}, nil
}
func (roleErrorStore) GetMemberRole(workspaceID, userID uint) (models.WorkspaceRole, error) {
	return "", assert.AnError
}

func TestCustomerService_LastServiceAtIncludesLastVisit(t *testing.T) {
	f := newCustomerFixture(t)
	visit := time.Date(2026, 11, 7, 14, 32, 0, 0, time.FixedZone("BRT", -3*3600))
	edited := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	onlyVisit := &models.Customer{WorkspaceID: fxWorkspaceID, Name: "Só visita", LastVisitAt: &visit, SourceUpdatedAt: &edited}
	require.NoError(t, f.db.Create(onlyVisit).Error)

	older := time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC)
	osNewer := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
	mixed := &models.Customer{WorkspaceID: fxWorkspaceID, Name: "OS mais nova", LastVisitAt: &older}
	require.NoError(t, f.db.Create(mixed).Error)
	require.NoError(t, f.db.Create(&models.ServiceRecord{WorkspaceID: fxWorkspaceID, SourceID: "1", CustomerID: &mixed.ID, OpenedAt: &osNewer}).Error)

	detail, err := f.svc.GetCustomer(fxOwner, onlyVisit.ID)
	require.NoError(t, err)
	require.NotNil(t, detail.LastServiceAt)
	assert.True(t, visit.Equal(*detail.LastServiceAt), "sem OS nem atendimento: usa a última visita")
	assert.Equal(t, time.UTC, detail.LastServiceAt.Location())
	body := jsonKeys(t, detail)
	assert.NotNil(t, body["last_visit_at"])
	assert.NotNil(t, body["source_updated_at"])

	detail, err = f.svc.GetCustomer(fxOwner, mixed.ID)
	require.NoError(t, err)
	assert.True(t, osNewer.Equal(*detail.LastServiceAt), "OS mais recente que a visita")

	page, err := f.svc.ListCustomers(fxManager, "", 1, 20)
	require.NoError(t, err)
	require.Len(t, page.Customers, 2)
	assert.Equal(t, onlyVisit.ID, page.Customers[0].ID, "ordem pela última visita")
	assert.Contains(t, jsonKeys(t, page.Customers[0]), "last_visit_at")
	assert.True(t, visit.Equal(*page.Customers[0].LastServiceAt))
}
