package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
)

type linkFixture struct {
	*vehicleFixture
	maria   *models.Customer
	joao    *models.Customer
	gol     *models.CustomerVehicle
	uno     *models.CustomerVehicle
	foreign *models.Customer
	alien   *models.CustomerVehicle
}

func newLinkFixture(t *testing.T) *linkFixture {
	t.Helper()
	vf := newVehicleFixture(t)
	vf.svc.WithCustomerLinks(repository.NewCustomerRepository(vf.db))
	f := &linkFixture{vehicleFixture: vf}

	f.maria = &models.Customer{WorkspaceID: fxWorkspaceID, Name: "Maria Souza", CPF: "12345678909", CEP: "01310100", Mobile: "11987654321"}
	f.joao = &models.Customer{WorkspaceID: fxWorkspaceID, Name: "João"}
	f.foreign = &models.Customer{WorkspaceID: fxOtherWorkspc, Name: "Outra oficina"}
	for _, c := range []*models.Customer{f.maria, f.joao, f.foreign} {
		require.NoError(t, vf.db.Create(c).Error)
	}
	f.gol = &models.CustomerVehicle{WorkspaceID: fxWorkspaceID, CustomerID: &f.maria.ID, Plate: "ABC1D23", Manufacturer: "VW", Model: "Gol 1.6"}
	f.uno = &models.CustomerVehicle{WorkspaceID: fxWorkspaceID, Plate: "UNO1A11", Model: "Uno"}
	f.alien = &models.CustomerVehicle{WorkspaceID: fxOtherWorkspc, CustomerID: &f.foreign.ID, Plate: "OUT1R00", Model: "Fusca"}
	for _, v := range []*models.CustomerVehicle{f.gol, f.uno, f.alien} {
		require.NoError(t, vf.db.Create(v).Error)
	}
	return f
}

func TestCreateList_VehicleLinkFillsDefaults(t *testing.T) {
	f := newLinkFixture(t)

	list, err := f.svc.CreateList(fxManager, VehicleInput{CustomerVehicleID: &f.gol.ID})
	require.NoError(t, err)
	got := f.reload(t, list.ID)
	assert.Equal(t, "VW Gol 1.6", got.Title)
	assert.Equal(t, "ABC1D23", got.Plate)
	require.NotNil(t, got.CustomerVehicleID)
	assert.Equal(t, f.gol.ID, *got.CustomerVehicleID)
	require.NotNil(t, got.CustomerID, "sem customer_id, usa o dono do veículo")
	assert.Equal(t, f.maria.ID, *got.CustomerID)
	assert.Equal(t, "Maria Souza", got.Customer)
	assert.Equal(t, "12345678909", got.CustomerCPF)
	assert.Equal(t, "01310100", got.CustomerCEP)
	assert.Equal(t, "11987654321", got.CustomerWhatsApp)

	// Vehicle sem dono: só placa/título; título só com modelo.
	list, err = f.svc.CreateList(fxOwner, VehicleInput{CustomerVehicleID: &f.uno.ID})
	require.NoError(t, err)
	got = f.reload(t, list.ID)
	assert.Equal(t, "Uno", got.Title)
	assert.Nil(t, got.CustomerID)
	assert.Equal(t, "", got.Customer)
}

func TestCreateList_CustomerLinkKeepsWhatWasTyped(t *testing.T) {
	f := newLinkFixture(t)
	list, err := f.svc.CreateList(fxOwner, VehicleInput{
		Title: "Gol do cliente", Plate: "xyz9w87", Customer: "Maria S.",
		CustomerCPF: ptr("529.982.247-25"), CustomerID: &f.maria.ID, CustomerVehicleID: &f.gol.ID,
	})
	require.NoError(t, err)
	got := f.reload(t, list.ID)
	assert.Equal(t, "Gol do cliente", got.Title)
	assert.Equal(t, "XYZ9W87", got.Plate)
	assert.Equal(t, "Maria S.", got.Customer)
	assert.Equal(t, "52998224725", got.CustomerCPF, "CPF digitado não é sobrescrito")
	assert.Equal(t, "01310100", got.CustomerCEP, "vazio é preenchido pelo cadastro")
	assert.Equal(t, f.maria.ID, *got.CustomerID)
}

func TestCreateList_InvalidLinks(t *testing.T) {
	f := newLinkFixture(t)
	cases := []struct {
		in   VehicleInput
		want error
	}{
		{VehicleInput{Title: "Gol", CustomerID: &f.foreign.ID}, ErrInvalidCustomer},
		{VehicleInput{Title: "Gol", CustomerID: ptr(uint(999))}, ErrInvalidCustomer},
		{VehicleInput{Title: "Gol", CustomerVehicleID: &f.alien.ID}, ErrInvalidCustomerVehicle},
		{VehicleInput{CustomerVehicleID: ptr(uint(999))}, ErrInvalidCustomerVehicle},
		{VehicleInput{Title: "Gol", CustomerID: &f.joao.ID, CustomerVehicleID: &f.gol.ID}, ErrVehicleCustomerMismatch},
		{VehicleInput{Title: "Uno", CustomerID: &f.joao.ID, CustomerVehicleID: &f.uno.ID}, ErrVehicleCustomerMismatch},
		{VehicleInput{CustomerVehicleID: ptr(uint(0))}, ErrTitleRequired},
	}
	for _, tc := range cases {
		_, err := f.svc.CreateList(fxOwner, tc.in)
		assert.ErrorIs(t, err, tc.want)
	}

	blank := &models.CustomerVehicle{WorkspaceID: fxWorkspaceID, Plate: "BLA1N00"}
	require.NoError(t, f.db.Create(blank).Error)
	_, err := f.svc.CreateList(fxOwner, VehicleInput{CustomerVehicleID: &blank.ID})
	assert.ErrorIs(t, err, ErrTitleRequired, "veículo sem modelo não preenche o título")

	var count int64
	f.db.Model(&models.TaskList{}).Count(&count)
	assert.Zero(t, count, "nada é criado com vínculo inválido")

	// Sem oficina ou sem cadastro ligado, qualquer id é inválido.
	_, err = f.svc.CreateList(fxOutsider, VehicleInput{Title: "Gol", CustomerID: &f.maria.ID})
	assert.ErrorIs(t, err, ErrInvalidCustomer)
	_, err = f.svc.CreateList(fxOutsider, VehicleInput{Title: "Gol", CustomerVehicleID: &f.gol.ID})
	assert.ErrorIs(t, err, ErrInvalidCustomerVehicle)
	unlinked := NewTaskListService(f.repo, repository.NewTaskItemRepository(f.db), fixtureWorkspaceStore())
	_, err = unlinked.CreateList(fxOwner, VehicleInput{Title: "Gol", CustomerID: &f.maria.ID})
	assert.ErrorIs(t, err, ErrInvalidCustomer)
}

func TestUpdateList_CustomerLinks(t *testing.T) {
	f := newLinkFixture(t)
	list, err := f.svc.CreateList(fxOwner, VehicleInput{CustomerVehicleID: &f.gol.ID})
	require.NoError(t, err)

	// PUT no estilo do app (só title/plate/customer) mantém vínculos e dados do cliente.
	_, err = f.svc.UpdateList(list.ID, fxManager, VehicleInput{Title: "Gol G5", Plate: "ABC1D23", Customer: "Maria Souza"})
	require.NoError(t, err)
	got := f.reload(t, list.ID)
	assert.Equal(t, f.maria.ID, *got.CustomerID)
	assert.Equal(t, f.gol.ID, *got.CustomerVehicleID)
	assert.Equal(t, "12345678909", got.CustomerCPF)
	assert.Equal(t, "Gol G5", got.Title)

	// Trocar o veículo para um sem dono, mandando customer_id = 0 (desvincula o cliente).
	_, err = f.svc.UpdateList(list.ID, fxOwner, VehicleInput{Title: "Uno", CustomerID: ptr(uint(0)), CustomerVehicleID: &f.uno.ID})
	require.NoError(t, err)
	got = f.reload(t, list.ID)
	assert.Nil(t, got.CustomerID)
	assert.Equal(t, f.uno.ID, *got.CustomerVehicleID)
	assert.Equal(t, "UNO1A11", got.Plate, "placa vazia no PUT é preenchida pelo veículo enviado")
	assert.Equal(t, "", got.Customer, "PUT substitui customer; sem cliente vinculado não há o que preencher")
}

func TestUpdateList_CustomerLinkDefaultsAndErrors(t *testing.T) {
	f := newLinkFixture(t)
	list, err := f.svc.CreateList(fxOwner, VehicleInput{Title: "Atendimento"})
	require.NoError(t, err)

	_, err = f.svc.UpdateList(list.ID, fxOwner, VehicleInput{CustomerVehicleID: &f.gol.ID})
	require.NoError(t, err)
	got := f.reload(t, list.ID)
	assert.Equal(t, "VW Gol 1.6", got.Title)
	assert.Equal(t, "ABC1D23", got.Plate)
	assert.Equal(t, "Maria Souza", got.Customer)
	assert.Equal(t, f.maria.ID, *got.CustomerID)

	_, err = f.svc.UpdateList(list.ID, fxOwner, VehicleInput{Title: "x", CustomerID: &f.foreign.ID})
	assert.ErrorIs(t, err, ErrInvalidCustomer)
	_, err = f.svc.UpdateList(list.ID, fxOwner, VehicleInput{Title: "x", CustomerID: &f.joao.ID, CustomerVehicleID: &f.gol.ID})
	assert.ErrorIs(t, err, ErrVehicleCustomerMismatch)
	_, err = f.svc.UpdateList(list.ID, fxOwner, VehicleInput{})
	assert.ErrorIs(t, err, ErrTitleRequired)
	blank := &models.CustomerVehicle{WorkspaceID: fxWorkspaceID, Plate: "BLA1N00"}
	require.NoError(t, f.db.Create(blank).Error)
	_, err = f.svc.UpdateList(list.ID, fxOwner, VehicleInput{CustomerVehicleID: &blank.ID})
	assert.ErrorIs(t, err, ErrTitleRequired)

	got = f.reload(t, list.ID)
	assert.Equal(t, f.maria.ID, *got.CustomerID, "erros não alteram o vínculo")
	assert.Equal(t, f.gol.ID, *got.CustomerVehicleID)
}

func TestCustomerLinks_MechanicSeesIDsButNoPersonalData(t *testing.T) {
	f := newLinkFixture(t)
	list, err := f.svc.CreateList(fxOwner, VehicleInput{CustomerVehicleID: &f.gol.ID})
	require.NoError(t, err)
	require.NoError(t, f.db.Create(&models.ListAssignment{TaskListID: list.ID, UserID: fxEditor, AssignedBy: fxOwner}).Error)

	got, err := f.svc.GetList(list.ID, fxEditor)
	require.NoError(t, err)
	assertNoCustomerData(t, got)
	body := jsonKeys(t, got)
	assert.Equal(t, float64(f.maria.ID), body["customer_id"])
	assert.Equal(t, float64(f.gol.ID), body["customer_vehicle_id"])

	page, err := f.svc.ListAll(fxEditor, "", 1, 20, "", true)
	require.NoError(t, err)
	require.Len(t, page.Lists, 1)
	assertNoCustomerData(t, page.Lists[0])
	assert.Equal(t, float64(f.gol.ID), jsonKeys(t, page.Lists[0])["customer_vehicle_id"])
}
