package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
)

type syncFixture struct {
	db    *gorm.DB
	repo  *repository.CustomerRepository
	runs  *repository.IntegrationRepository
	svc   *OnMotorSyncService
	token *models.IntegrationToken
	clock time.Time
}

func newSyncFixture(t *testing.T) *syncFixture {
	t.Helper()
	db := setupTestDB(t)
	f := &syncFixture{
		db:    db,
		repo:  repository.NewCustomerRepository(db),
		runs:  repository.NewIntegrationRepository(db),
		token: &models.IntegrationToken{ID: 1, WorkspaceID: fxWorkspaceID},
		clock: time.Date(2026, 11, 9, 18, 0, 0, 0, time.UTC),
	}
	f.svc = NewOnMotorSyncService(f.repo, f.runs, fixtureWorkspaceStore())
	f.svc.now = func() time.Time { return f.clock }
	return f
}

func (f *syncFixture) importBatch(t *testing.T, in ImportInput) *ImportResult {
	t.Helper()
	res, err := f.svc.ImportWithToken(f.token, in)
	require.NoError(t, err)
	return res
}

func (f *syncFixture) customerBySource(t *testing.T, sourceID string) *models.Customer {
	t.Helper()
	c, err := f.repo.FindCustomerBySource(fxWorkspaceID, models.SourceOnMotor, sourceID)
	require.NoError(t, err)
	require.NotNil(t, c, "cliente OnMotor "+sourceID)
	return c
}

func (f *syncFixture) vehicleBySource(t *testing.T, sourceID string) *models.CustomerVehicle {
	t.Helper()
	v, err := f.repo.FindVehicleBySource(fxWorkspaceID, models.SourceOnMotor, sourceID)
	require.NoError(t, err)
	require.NotNil(t, v, "veículo OnMotor "+sourceID)
	return v
}

func (f *syncFixture) serviceBySource(t *testing.T, sourceID string) *models.ServiceRecord {
	t.Helper()
	sr, err := f.repo.FindServiceRecordBySource(fxWorkspaceID, models.SourceOnMotor, sourceID)
	require.NoError(t, err)
	require.NotNil(t, sr, "OS OnMotor "+sourceID)
	return sr
}

func decodeImport(t *testing.T, body string) ImportInput {
	t.Helper()
	var in ImportInput
	require.NoError(t, json.Unmarshal([]byte(body), &in))
	return in
}

const fullImportBody = `{
  "trigger": "schedule",
  "customers": [
    {"source_id": 123, "name": "Maria Souza", "person_type": "F", "cpf": "123.456.789-09", "rg": "12.345.678-9",
     "phone": "(11) 3333-4444", "mobile": "11 98765-4321", "email": "Maria@Exemplo.com", "cep": "01310-100",
     "street": "Av. Paulista", "number": 1000, "district": "Bela Vista", "city": "São Paulo", "uf": "sp",
     "birth_date": "03/05/1980", "notes": "cliente antiga"},
    {"source_id": "124", "name": "Oficina Parceira Ltda", "person_type": "J", "cnpj": "11.222.333/0001-81"}
  ],
  "vehicles": [
    {"source_id": "889", "customer_source_id": 123, "plate": "abc-1d23", "manufacturer": "VW", "model": "Gol 1.6",
     "year_fab": "2014", "year_model": 2015, "doors": 4, "color": "Prata", "last_mileage": "80.000"},
    {"source_id": "890", "customer_source_id": "124", "plate": "XYZ9W87", "model": "Strada"}
  ],
  "services": [
    {"source_id": "5001", "customer_source_id": "123", "vehicle_source_id": "889", "mileage": 85000,
     "type": "Revisão", "status": "Entregue", "opened_at": "10/03/2026 09:30", "delivered_at": "2026-03-12T17:00:00-03:00",
     "notes": "", "items": [{"kind":"servico","description":"Troca de óleo","quantity":1,"unit_price":120,"total":120}],
     "total": "120,00"},
    {"source_id": "5002", "plate": "XYZ-9W87", "opened_at": "2026-04-01", "status": "Aberta"}
  ],
  "final": true
}`

func TestImport_CreatesLinksAndClosesRun(t *testing.T) {
	f := newSyncFixture(t)
	res := f.importBatch(t, decodeImport(t, fullImportBody))

	assert.Equal(t, models.SyncStatusSuccess, res.Status)
	assert.Equal(t, SyncCounters{CustomersCreated: 2, VehiclesCreated: 2, ServicesCreated: 2}, res.Counters)
	assert.Empty(t, res.Warnings)

	run, err := f.runs.FindRun(fxWorkspaceID, res.RunID)
	require.NoError(t, err)
	assert.Equal(t, models.SyncTriggerSchedule, run.Trigger)
	require.NotNil(t, run.FinishedAt)

	maria := f.customerBySource(t, "123")
	assert.Equal(t, "Maria Souza", maria.Name)
	assert.Equal(t, "pf", maria.PersonType)
	assert.Equal(t, "12345678909", maria.CPF)
	assert.Equal(t, "1133334444", maria.Phone)
	assert.Equal(t, "11987654321", maria.Mobile)
	assert.Equal(t, "maria@exemplo.com", maria.Email)
	assert.Equal(t, "01310100", maria.CEP)
	assert.Equal(t, "SP", maria.UF)
	assert.Equal(t, "1000", maria.Number)
	assert.Equal(t, "1980-05-03", maria.BirthDate.String())
	require.NotNil(t, maria.SyncedAt)
	parceira := f.customerBySource(t, "124")
	assert.Equal(t, "pj", parceira.PersonType)
	assert.Equal(t, "11222333000181", parceira.CNPJ)

	gol := f.vehicleBySource(t, "889")
	assert.Equal(t, "ABC1D23", gol.Plate)
	assert.Equal(t, maria.ID, *gol.CustomerID)
	assert.Equal(t, 2014, *gol.YearFab)
	assert.Equal(t, 2015, *gol.YearModel)
	assert.Equal(t, 85000, *gol.LastMileage, "km da OS maior que o do cadastro")

	os1 := f.serviceBySource(t, "5001")
	assert.Equal(t, maria.ID, *os1.CustomerID)
	assert.Equal(t, gol.ID, *os1.CustomerVehicleID)
	assert.Equal(t, "ABC1D23", os1.Plate, "placa vem do veículo quando a OS não traz")
	assert.True(t, time.Date(2026, 3, 10, 12, 30, 0, 0, time.UTC).Equal(*os1.OpenedAt))
	assert.True(t, time.Date(2026, 3, 12, 20, 0, 0, 0, time.UTC).Equal(*os1.DeliveredAt))
	assert.Equal(t, 120.0, *os1.Total)
	assert.JSONEq(t, `[{"kind":"servico","description":"Troca de óleo","quantity":1,"unit_price":120,"total":120}]`, string(os1.Items))

	os2 := f.serviceBySource(t, "5002")
	strada := f.vehicleBySource(t, "890")
	assert.Equal(t, strada.ID, *os2.CustomerVehicleID, "sem vehicle_source_id: acha pela placa")
	assert.Equal(t, parceira.ID, *os2.CustomerID, "cliente herdado do veículo")
	assert.Equal(t, "[]", string(os2.Items))
}

func TestImport_RerunIsIdempotent(t *testing.T) {
	f := newSyncFixture(t)
	f.importBatch(t, decodeImport(t, fullImportBody))
	maria := f.customerBySource(t, "123")
	gol := f.vehicleBySource(t, "889")
	os1 := f.serviceBySource(t, "5001")

	f.clock = f.clock.Add(24 * time.Hour)
	res := f.importBatch(t, decodeImport(t, fullImportBody))
	assert.Equal(t, SyncCounters{}, res.Counters, "nada mudou: nada criado nem atualizado")
	assert.Equal(t, models.SyncStatusSuccess, res.Status)

	again := f.customerBySource(t, "123")
	assert.True(t, maria.UpdatedAt.Equal(again.UpdatedAt), "updated_at não muda sem mudança real")
	assert.True(t, f.clock.Equal(*again.SyncedAt), "synced_at registra a nova sincronização")
	assert.True(t, gol.UpdatedAt.Equal(f.vehicleBySource(t, "889").UpdatedAt))
	assert.True(t, os1.UpdatedAt.Equal(f.serviceBySource(t, "5001").UpdatedAt))
	assert.True(t, f.clock.Equal(*f.serviceBySource(t, "5001").SyncedAt))

	for model, want := range map[any]int64{&models.Customer{}: 2, &models.CustomerVehicle{}: 2, &models.ServiceRecord{}: 2} {
		var n int64
		f.db.Model(model).Count(&n)
		assert.Equal(t, want, n, "sem duplicatas")
	}

	// Mudança real em cada entidade conta como atualização.
	res = f.importBatch(t, decodeImport(t, `{"customers":[{"source_id":"123","city":"Campinas"}],
		"vehicles":[{"source_id":"889","color":"Preto"}],
		"services":[{"source_id":"5001","status":"Arquivada"}],"final":true}`))
	assert.Equal(t, SyncCounters{CustomersUpdated: 1, VehiclesUpdated: 1, ServicesUpdated: 1}, res.Counters)
	updated := f.customerBySource(t, "123")
	assert.Equal(t, "Campinas", updated.City)
	assert.Equal(t, "12345678909", updated.CPF, "campo ausente mantém o valor")
	assert.Equal(t, "Maria Souza", updated.Name)
}

func TestImport_NeverDeletes(t *testing.T) {
	f := newSyncFixture(t)
	f.importBatch(t, decodeImport(t, fullImportBody))
	f.importBatch(t, decodeImport(t, `{"customers":[{"source_id":"124","name":"Oficina Parceira Ltda"}],"final":true}`))
	f.customerBySource(t, "123")
	f.vehicleBySource(t, "889")
	f.serviceBySource(t, "5001")
}

func TestImport_BatchesAccumulateUntilFinal(t *testing.T) {
	f := newSyncFixture(t)
	first := f.importBatch(t, decodeImport(t, `{"trigger":"manual","customers":[{"source_id":"1","name":"A"},{"source_id":"2","name":"B"}]}`))
	assert.Equal(t, models.SyncStatusRunning, first.Status)
	assert.NotZero(t, first.RunID)

	runID := first.RunID
	second := f.importBatch(t, ImportInput{RunID: &runID, Customers: []ImportCustomer{{SourceID: Flex("3"), Name: Flex("C")}}})
	assert.Equal(t, runID, second.RunID)
	assert.Equal(t, 3, second.Counters.CustomersCreated)
	assert.Equal(t, models.SyncStatusRunning, second.Status)

	last := f.importBatch(t, ImportInput{RunID: &runID, Vehicles: []ImportVehicle{{SourceID: Flex("v1")}}, Final: true})
	assert.Equal(t, models.SyncStatusSuccess, last.Status)
	assert.Equal(t, 1, last.Counters.VehiclesCreated)
	run, err := f.runs.FindRun(fxWorkspaceID, runID)
	require.NoError(t, err)
	assert.Equal(t, models.SyncTriggerManual, run.Trigger)
	assert.Equal(t, 3, run.CustomersCreated)
	require.NotNil(t, run.FinishedAt)

	_, err = f.svc.ImportWithToken(f.token, ImportInput{RunID: &runID})
	assert.ErrorIs(t, err, ErrSyncRunFinished)
	missing := uint(999)
	_, err = f.svc.ImportWithToken(f.token, ImportInput{RunID: &missing})
	assert.ErrorIs(t, err, ErrSyncRunNotFound)
	foreignToken := &models.IntegrationToken{ID: 2, WorkspaceID: fxOtherWorkspc}
	open := f.importBatch(t, ImportInput{})
	_, err = f.svc.ImportWithToken(foreignToken, ImportInput{RunID: &open.RunID})
	assert.ErrorIs(t, err, ErrSyncRunNotFound, "run de outra oficina")
}

func TestImport_MergesWithLocalRecords(t *testing.T) {
	f := newSyncFixture(t)
	local := &models.Customer{WorkspaceID: fxWorkspaceID, Name: "Maria (balcão)", Source: models.SourceLocal, CPF: "12345678909", Notes: "cadastro local"}
	require.NoError(t, f.db.Create(local).Error)
	localCar := &models.CustomerVehicle{WorkspaceID: fxWorkspaceID, CustomerID: &local.ID, Source: models.SourceLocal, Plate: "ABC1D23", Color: "Prata"}
	require.NoError(t, f.db.Create(localCar).Error)

	res := f.importBatch(t, decodeImport(t, `{
		"customers":[{"source_id":"123","name":"Maria Souza","cpf":"123.456.789-09"}],
		"vehicles":[{"source_id":"889","customer_source_id":"123","plate":"ABC-1D23","model":"Gol"}],
		"final":true}`))
	assert.Equal(t, SyncCounters{CustomersUpdated: 1, VehiclesUpdated: 1}, res.Counters, "funde em vez de duplicar")

	merged := f.customerBySource(t, "123")
	assert.Equal(t, local.ID, merged.ID)
	assert.Equal(t, models.SourceOnMotor, merged.Source)
	assert.Equal(t, "Maria Souza", merged.Name)
	assert.Equal(t, "cadastro local", merged.Notes, "campos ausentes no OnMotor ficam")
	car := f.vehicleBySource(t, "889")
	assert.Equal(t, localCar.ID, car.ID)
	assert.Equal(t, "Gol", car.Model)
	assert.Equal(t, "Prata", car.Color)

	var n int64
	f.db.Model(&models.Customer{}).Count(&n)
	assert.Equal(t, int64(1), n)
}

func TestImport_InvalidFieldsAreBlankedWithWarnings(t *testing.T) {
	f := newSyncFixture(t)
	res := f.importBatch(t, decodeImport(t, `{
		"customers":[{"source_id":"77","name":"João da Silva","cpf":"111.111.111-11","cnpj":"123","phone":"12",
			"mobile":"999","email":"joao-arroba-x","email_secondary":"x@","cep":"1234","uf":"São Paulo",
			"person_type":"outro","birth_date":"31/02/1990","city":"Santos"}],
		"vehicles":[{"source_id":"v77","customer_source_id":"77","plate":"JOA0S11","year_fab":"abc","year_model":1800,
			"doors":99,"last_mileage":"-5"}],
		"services":[{"source_id":"os77","vehicle_source_id":"v77","mileage":"muito","opened_at":"ontem",
			"diagnosed_at":"x","items":{"oops":true},"total":"caro"}],
		"final":true}`))

	assert.Equal(t, models.SyncStatusPartial, res.Status)
	assert.Equal(t, 1, res.Counters.CustomersCreated, "registro gravado mesmo com campos inválidos")
	assert.Equal(t, 1, res.Counters.VehiclesCreated)
	assert.Equal(t, 1, res.Counters.ServicesCreated)
	assert.Zero(t, res.Counters.Skipped)

	c := f.customerBySource(t, "77")
	for field, v := range map[string]string{"cpf": c.CPF, "cnpj": c.CNPJ, "phone": c.Phone, "mobile": c.Mobile, "email": c.Email, "email2": c.EmailSecondary, "cep": c.CEP, "uf": c.UF, "pt": c.PersonType} {
		assert.Equal(t, "", v, field)
	}
	assert.Nil(t, c.BirthDate)
	assert.Equal(t, "Santos", c.City)
	v := f.vehicleBySource(t, "v77")
	assert.Nil(t, v.YearFab)
	assert.Nil(t, v.YearModel)
	assert.Nil(t, v.Doors)
	assert.Nil(t, v.LastMileage)
	sr := f.serviceBySource(t, "os77")
	assert.Nil(t, sr.Mileage)
	assert.Nil(t, sr.OpenedAt)
	assert.Nil(t, sr.Total)
	assert.Equal(t, "[]", string(sr.Items))

	assert.Len(t, res.Warnings, 19)
	assert.Contains(t, res.Warnings, "cliente OnMotor 77: CPF inválido (campo ignorado)")
	assert.Contains(t, res.Warnings, "OS OnMotor os77: items inválido (esperado um array; campo ignorado)")
	assertNoPersonalData(t, res.Warnings, "João", "Silva", "111.111.111-11", "11111111111", "joao", "JOA0S11", "Santos", "999")
}

func assertNoPersonalData(t *testing.T, warnings []string, secrets ...string) {
	t.Helper()
	joined := strings.Join(warnings, "\n")
	for _, s := range secrets {
		assert.NotContains(t, joined, s, "aviso não pode conter dado pessoal")
	}
}

func TestImport_SkipsRecordsThatCannotBeStored(t *testing.T) {
	f := newSyncFixture(t)
	res := f.importBatch(t, decodeImport(t, `{
		"customers":[{"name":"Sem ID"},{"source_id":"  ","name":"Branco"},{"source_id":"9"},{"source_id":"10","name":"Ok"}],
		"vehicles":[{"plate":"AAA1A11"}],
		"services":[{"status":"x"}],
		"final":true}`))
	assert.Equal(t, 5, res.Counters.Skipped)
	assert.Equal(t, 1, res.Counters.CustomersCreated)
	assert.Equal(t, models.SyncStatusPartial, res.Status)
	assert.Contains(t, res.Warnings, "cliente na posição 1 sem source_id (ignorado)")
	assert.Contains(t, res.Warnings, "cliente OnMotor 9 sem nome (ignorado)")
	assert.Contains(t, res.Warnings, "veículo na posição 1 sem source_id (ignorado)")
	assert.Contains(t, res.Warnings, "OS na posição 1 sem source_id (ignorada)")
	assertNoPersonalData(t, res.Warnings, "Sem ID", "Branco", "AAA1A11")

	// Cliente já importado não perde o nome se o OnMotor mandar vazio.
	res = f.importBatch(t, decodeImport(t, `{"customers":[{"source_id":"10","name":""}],"final":true}`))
	assert.Zero(t, res.Counters.Skipped)
	assert.Equal(t, "Ok", f.customerBySource(t, "10").Name)
}

func TestImport_DuplicateCPFKeepsFirst(t *testing.T) {
	f := newSyncFixture(t)
	res := f.importBatch(t, decodeImport(t, `{"customers":[
		{"source_id":"123","name":"Maria","cpf":"12345678909"},
		{"source_id":"456","name":"Maria Duplicada","cpf":"123.456.789-09"}],"final":true}`))
	assert.Equal(t, 2, res.Counters.CustomersCreated)
	assert.Equal(t, models.SyncStatusPartial, res.Status)
	assert.Equal(t, []string{"CPF duplicado em 2 clientes (OnMotor 123, cliente OnMotor 456); mantido no primeiro"}, []string(res.Warnings))
	assert.Equal(t, "12345678909", f.customerBySource(t, "123").CPF)
	assert.Equal(t, "", f.customerBySource(t, "456").CPF)
	assertNoPersonalData(t, res.Warnings, "Maria", "12345678909")

	// Rodar de novo é estável.
	res = f.importBatch(t, decodeImport(t, `{"customers":[
		{"source_id":"456","name":"Maria Duplicada","cpf":"123.456.789-09"},
		{"source_id":"123","name":"Maria","cpf":"12345678909"}],"final":true}`))
	assert.Equal(t, SyncCounters{}, res.Counters)
	assert.Equal(t, "12345678909", f.customerBySource(t, "123").CPF)

	// CPF de cliente local já vinculado ao OnMotor por outro id também não é tomado.
	local := &models.Customer{WorkspaceID: fxWorkspaceID, Name: "Local", CPF: "52998224725"}
	require.NoError(t, f.db.Create(local).Error)
	res = f.importBatch(t, decodeImport(t, `{"customers":[{"source_id":"123","cpf":"529.982.247-25"}],"final":true}`))
	assert.Contains(t, res.Warnings[0], fmt.Sprintf("local #%d", local.ID))
	assert.Equal(t, "", f.customerBySource(t, "123").CPF)
}

func TestImport_DuplicatePlateLastWriteWins(t *testing.T) {
	f := newSyncFixture(t)
	res := f.importBatch(t, decodeImport(t, `{"vehicles":[
		{"source_id":"1","plate":"ABC1D23","model":"Gol"},
		{"source_id":"2","plate":"abc-1d23","model":"Gol novo dono"}],"final":true}`))
	assert.Equal(t, 2, res.Counters.VehiclesCreated)
	assert.Equal(t, []string{"placa duplicada: veículo OnMotor 2 assumiu a placa do veículo OnMotor 1"}, []string(res.Warnings))
	assert.Equal(t, "", f.vehicleBySource(t, "1").Plate)
	assert.Equal(t, "ABC1D23", f.vehicleBySource(t, "2").Plate)

	local := &models.CustomerVehicle{WorkspaceID: fxWorkspaceID, Plate: "LOC4L00", Source: models.SourceLocal}
	require.NoError(t, f.db.Create(local).Error)
	res = f.importBatch(t, decodeImport(t, `{"vehicles":[{"source_id":"2","plate":"LOC4L00"}],"final":true}`))
	assert.Contains(t, res.Warnings, fmt.Sprintf("placa duplicada: veículo OnMotor 2 assumiu a placa do veículo local #%d", local.ID))
	assert.Equal(t, 1, res.Counters.VehiclesUpdated)
}

func TestImport_ServiceLinksAndMileage(t *testing.T) {
	f := newSyncFixture(t)
	f.importBatch(t, decodeImport(t, `{"customers":[{"source_id":"c1","name":"A"}],
		"vehicles":[{"source_id":"v1","customer_source_id":"c1","plate":"AAA1A11","last_mileage":80000},
		            {"source_id":"v2","customer_source_id":"desconhecido","plate":"BBB2B22"}],"final":true}`))
	assert.Nil(t, f.vehicleBySource(t, "v2").CustomerID)

	res := f.importBatch(t, decodeImport(t, `{"services":[
		{"source_id":"s1","vehicle_source_id":"v1","mileage":70000},
		{"source_id":"s2","vehicle_source_id":"nao-existe","plate":"AAA-1A11","customer_source_id":"nao-existe","mileage":"90.500"},
		{"source_id":"s3","plate":"ZZZ9Z99"}],"final":true}`))
	assert.Equal(t, 3, res.Counters.ServicesCreated)
	assert.Contains(t, res.Warnings, "OS OnMotor s2: veículo OnMotor nao-existe não encontrado")
	assert.Contains(t, res.Warnings, "OS OnMotor s2: cliente OnMotor nao-existe não encontrado")

	v1 := f.vehicleBySource(t, "v1")
	assert.Equal(t, 90500, *v1.LastMileage, "maior km entre cadastro e OS")
	s2 := f.serviceBySource(t, "s2")
	assert.Equal(t, v1.ID, *s2.CustomerVehicleID, "fallback pela placa")
	assert.Equal(t, *v1.CustomerID, *s2.CustomerID)
	assert.Equal(t, "AAA1A11", s2.Plate)
	s3 := f.serviceBySource(t, "s3")
	assert.Nil(t, s3.CustomerVehicleID)
	assert.Equal(t, "ZZZ9Z99", s3.Plate)

	// Vehicle com km menor não reduz last_mileage.
	f.importBatch(t, decodeImport(t, `{"vehicles":[{"source_id":"v1","last_mileage":1000}],"final":true}`))
	assert.Equal(t, 90500, *f.vehicleBySource(t, "v1").LastMileage)
}

func TestImport_LimitsTriggerAndPermissions(t *testing.T) {
	f := newSyncFixture(t)
	tooMany := ImportInput{Customers: make([]ImportCustomer, MaxImportRecords+1)}
	_, err := f.svc.ImportWithToken(f.token, tooMany)
	assert.ErrorIs(t, err, ErrImportBatchTooLarge)
	_, err = f.svc.ImportWithToken(f.token, ImportInput{Services: make([]ImportService, MaxImportRecords+1)})
	assert.ErrorIs(t, err, ErrImportBatchTooLarge)
	_, err = f.svc.ImportWithToken(f.token, ImportInput{Trigger: "upload"})
	assert.ErrorIs(t, err, ErrInvalidTrigger, "upload só pelo painel")

	res := f.importBatch(t, ImportInput{})
	run, _ := f.runs.FindRun(fxWorkspaceID, res.RunID)
	assert.Equal(t, models.SyncTriggerSchedule, run.Trigger, "padrão do robô")

	manual, err := f.svc.ImportManual(fxManager, ImportInput{Customers: []ImportCustomer{{SourceID: Flex("1"), Name: Flex("A")}}, Final: true})
	require.NoError(t, err)
	run, _ = f.runs.FindRun(fxWorkspaceID, manual.RunID)
	assert.Equal(t, models.SyncTriggerUpload, run.Trigger)
	assert.Equal(t, models.SyncStatusSuccess, run.Status)

	_, err = f.svc.ImportManual(fxEditor, ImportInput{})
	assert.ErrorIs(t, err, ErrIntegrationForbidden)
	_, err = f.svc.ImportManual(fxOutsider, ImportInput{})
	assert.ErrorIs(t, err, ErrNoWorkspace)
}

func TestImport_FailRunAndListRuns(t *testing.T) {
	f := newSyncFixture(t)
	open := f.importBatch(t, ImportInput{})
	run, err := f.svc.FailRun(f.token, open.RunID, "  "+strings.Repeat("x", 600))
	require.NoError(t, err)
	assert.Equal(t, models.SyncStatusError, run.Status)
	assert.Len(t, run.Error, maxSyncErrorLength)
	require.NotNil(t, run.FinishedAt)

	_, err = f.svc.FailRun(f.token, open.RunID, "de novo")
	assert.ErrorIs(t, err, ErrSyncRunFinished)
	_, err = f.svc.FailRun(f.token, 999, "x")
	assert.ErrorIs(t, err, ErrSyncRunNotFound)

	f.clock = f.clock.Add(time.Minute)
	second := f.importBatch(t, ImportInput{})
	run, err = f.svc.FailRun(f.token, second.RunID, "")
	require.NoError(t, err)
	assert.Equal(t, "falha informada pelo robô", run.Error)

	f.clock = f.clock.Add(time.Minute)
	f.importBatch(t, ImportInput{Final: true})

	runs, err := f.svc.ListRuns(fxOwner, 0)
	require.NoError(t, err)
	require.Len(t, runs, 3)
	assert.Equal(t, models.SyncStatusSuccess, runs[0].Status, "mais recente primeiro")
	runs, err = f.svc.ListRuns(fxManager, 1)
	require.NoError(t, err)
	assert.Len(t, runs, 1)
	runs, err = f.svc.ListRuns(fxManager, 500)
	require.NoError(t, err)
	assert.Len(t, runs, 3)
	_, err = f.svc.ListRuns(fxEditor, 10)
	assert.ErrorIs(t, err, ErrIntegrationForbidden)
}

func TestImport_WarningsAreCapped(t *testing.T) {
	f := newSyncFixture(t)
	customers := make([]ImportCustomer, 250)
	for i := range customers {
		customers[i] = ImportCustomer{SourceID: Flex(fmt.Sprint(i + 1)), Name: Flex("X"), CEP: Flex("1")}
	}
	res := f.importBatch(t, ImportInput{Customers: customers, Final: true})
	assert.Len(t, res.Warnings, maxSyncWarnings+1)
	assert.Equal(t, warningsTruncated, res.Warnings[maxSyncWarnings])
	assert.Equal(t, 250, res.Counters.CustomersCreated)
}

func TestSyncRunJSONHasNoPersonalData(t *testing.T) {
	f := newSyncFixture(t)
	res := f.importBatch(t, decodeImport(t, fullImportBody))
	run, err := f.runs.FindRun(fxWorkspaceID, res.RunID)
	require.NoError(t, err)
	raw, err := json.Marshal(run)
	require.NoError(t, err)
	for _, secret := range []string{"Maria", "12345678909", "11987654321", "ABC1D23", "maria@exemplo.com"} {
		assert.NotContains(t, string(raw), secret)
	}
}

// failingStore delega ao banco real e falha no método escolhido.
type failingStore struct {
	OnMotorStore
	fail string
}

func (s *failingStore) err(method string) error {
	if s.fail == method {
		return assert.AnError
	}
	return nil
}

func (s *failingStore) CreateCustomer(c *models.Customer) error {
	if err := s.err("CreateCustomer"); err != nil {
		return err
	}
	return s.OnMotorStore.CreateCustomer(c)
}
func (s *failingStore) SaveCustomer(c *models.Customer) error {
	if err := s.err("SaveCustomer"); err != nil {
		return err
	}
	return s.OnMotorStore.SaveCustomer(c)
}
func (s *failingStore) FindCustomerByCPF(ws uint, cpf string) (*models.Customer, error) {
	if err := s.err("FindCustomerByCPF"); err != nil {
		return nil, err
	}
	return s.OnMotorStore.FindCustomerByCPF(ws, cpf)
}
func (s *failingStore) FindCustomerBySource(ws uint, source, id string) (*models.Customer, error) {
	if err := s.err("FindCustomerBySource"); err != nil {
		return nil, err
	}
	return s.OnMotorStore.FindCustomerBySource(ws, source, id)
}
func (s *failingStore) CreateVehicle(v *models.CustomerVehicle) error {
	if err := s.err("CreateVehicle"); err != nil {
		return err
	}
	return s.OnMotorStore.CreateVehicle(v)
}
func (s *failingStore) SaveVehicle(v *models.CustomerVehicle) error {
	if err := s.err("SaveVehicle"); err != nil {
		return err
	}
	return s.OnMotorStore.SaveVehicle(v)
}
func (s *failingStore) FindVehicleByPlate(ws uint, plate string) (*models.CustomerVehicle, error) {
	if err := s.err("FindVehicleByPlate"); err != nil {
		return nil, err
	}
	return s.OnMotorStore.FindVehicleByPlate(ws, plate)
}
func (s *failingStore) FindVehicleBySource(ws uint, source, id string) (*models.CustomerVehicle, error) {
	if err := s.err("FindVehicleBySource"); err != nil {
		return nil, err
	}
	return s.OnMotorStore.FindVehicleBySource(ws, source, id)
}
func (s *failingStore) ClearVehiclePlate(id uint) error {
	if err := s.err("ClearVehiclePlate"); err != nil {
		return err
	}
	return s.OnMotorStore.ClearVehiclePlate(id)
}
func (s *failingStore) CreateServiceRecord(sr *models.ServiceRecord) error {
	if err := s.err("CreateServiceRecord"); err != nil {
		return err
	}
	return s.OnMotorStore.CreateServiceRecord(sr)
}
func (s *failingStore) SaveServiceRecord(sr *models.ServiceRecord) error {
	if err := s.err("SaveServiceRecord"); err != nil {
		return err
	}
	return s.OnMotorStore.SaveServiceRecord(sr)
}
func (s *failingStore) TouchServiceRecordSynced(id uint, at time.Time) error {
	if err := s.err("TouchServiceRecordSynced"); err != nil {
		return err
	}
	return s.OnMotorStore.TouchServiceRecordSynced(id, at)
}
func (s *failingStore) FindServiceRecordBySource(ws uint, source, id string) (*models.ServiceRecord, error) {
	if err := s.err("FindServiceRecordBySource"); err != nil {
		return nil, err
	}
	return s.OnMotorStore.FindServiceRecordBySource(ws, source, id)
}

type failingRuns struct {
	SyncRunStore
	createErr, saveErr error
}

func (r *failingRuns) CreateRun(run *models.SyncRun) error {
	if r.createErr != nil {
		return r.createErr
	}
	return r.SyncRunStore.CreateRun(run)
}
func (r *failingRuns) SaveRun(run *models.SyncRun) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	return r.SyncRunStore.SaveRun(run)
}

func TestImport_StoreErrorsAbortTheBatch(t *testing.T) {
	// Base já importada, para exercitar também os caminhos de atualização.
	seed := `{"customers":[{"source_id":"123","name":"Maria","cpf":"12345678909"}],
		"vehicles":[{"source_id":"889","plate":"ABC1D23"},{"source_id":"890","plate":"XYZ9W87"}],
		"services":[{"source_id":"5001","vehicle_source_id":"889"}],"final":true}`
	batches := map[string]string{
		"FindCustomerBySource":      `{"customers":[{"source_id":"1","name":"A"}]}`,
		"FindCustomerByCPF":         `{"customers":[{"source_id":"1","name":"A","cpf":"52998224725"}]}`,
		"CreateCustomer":            `{"customers":[{"source_id":"1","name":"A"}]}`,
		"SaveCustomer":              `{"customers":[{"source_id":"123","name":"Maria Souza"}]}`,
		"FindVehicleBySource":       `{"vehicles":[{"source_id":"1"}]}`,
		"FindVehicleByPlate":        `{"vehicles":[{"source_id":"1","plate":"NEW1A11"}]}`,
		"CreateVehicle":             `{"vehicles":[{"source_id":"1"}]}`,
		"SaveVehicle":               `{"vehicles":[{"source_id":"889","color":"Azul"}]}`,
		"ClearVehiclePlate":         `{"vehicles":[{"source_id":"890","plate":"ABC1D23"}]}`,
		"FindServiceRecordBySource": `{"services":[{"source_id":"1"}]}`,
		"CreateServiceRecord":       `{"services":[{"source_id":"1"}]}`,
		"SaveServiceRecord":         `{"services":[{"source_id":"5001","status":"x"}]}`,
		"TouchServiceRecordSynced":  `{"services":[{"source_id":"5001"}]}`,
	}
	for method, body := range batches {
		f := newSyncFixture(t)
		f.importBatch(t, decodeImport(t, seed))
		f.svc.store = &failingStore{OnMotorStore: f.repo, fail: method}
		_, err := f.svc.ImportWithToken(f.token, decodeImport(t, body))
		assert.ErrorIs(t, err, assert.AnError, method)
	}

	// Falhas na busca do dono / do veículo da OS também abortam.
	for _, tc := range []struct{ method, body string }{
		{"FindCustomerBySource", `{"vehicles":[{"source_id":"1","customer_source_id":"123"}]}`},
		{"FindCustomerBySource", `{"services":[{"source_id":"1","customer_source_id":"123"}]}`},
		{"FindVehicleBySource", `{"services":[{"source_id":"1","vehicle_source_id":"889"}]}`},
	} {
		f := newSyncFixture(t)
		f.importBatch(t, decodeImport(t, seed))
		f.svc.store = &failOnLookup{failingStore{OnMotorStore: f.repo, fail: tc.method}}
		_, err := f.svc.ImportWithToken(f.token, decodeImport(t, tc.body))
		assert.ErrorIs(t, err, assert.AnError, tc.method+" "+tc.body)
	}

	f := newSyncFixture(t)
	f.svc.runs = &failingRuns{SyncRunStore: f.runs, createErr: assert.AnError}
	_, err := f.svc.ImportWithToken(f.token, ImportInput{})
	assert.ErrorIs(t, err, assert.AnError)
	f.svc.runs = &failingRuns{SyncRunStore: f.runs, saveErr: assert.AnError}
	_, err = f.svc.ImportWithToken(f.token, ImportInput{})
	assert.ErrorIs(t, err, assert.AnError)

	f.svc.runs = f.runs
	open := f.importBatch(t, ImportInput{})
	f.svc.runs = &failingRuns{SyncRunStore: f.runs, saveErr: assert.AnError}
	_, err = f.svc.FailRun(f.token, open.RunID, "x")
	assert.ErrorIs(t, err, assert.AnError)
}

// failOnLookup falha só na busca do dono (cliente 123) ou do veículo (889) referenciados pelo registro.
type failOnLookup struct {
	failingStore
}

func (s *failOnLookup) FindCustomerBySource(ws uint, source, id string) (*models.Customer, error) {
	if s.fail == "FindCustomerBySource" && id == "123" {
		return nil, assert.AnError
	}
	return s.OnMotorStore.FindCustomerBySource(ws, source, id)
}

func (s *failOnLookup) FindVehicleBySource(ws uint, source, id string) (*models.CustomerVehicle, error) {
	if s.fail == "FindVehicleBySource" && id == "889" {
		return nil, assert.AnError
	}
	return s.OnMotorStore.FindVehicleBySource(ws, source, id)
}

func TestDuplicateError_Message(t *testing.T) {
	err := &DuplicateError{Err: ErrPlateTaken, ExistingID: 3}
	assert.Equal(t, "placa já cadastrada em outro veículo", err.Error())
}
