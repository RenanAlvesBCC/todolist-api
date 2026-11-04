package services

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
)

const (
	fxOwner        uint = 1
	fxManager      uint = 2
	fxEditor       uint = 3
	fxEditorOther  uint = 4
	fxOutsider     uint = 9
	fxWorkspaceID  uint = 10
	fxOtherWorkspc uint = 20
)

type vehicleFixture struct {
	db   *gorm.DB
	repo *repository.TaskListRepository
	svc  *TaskListService
}

func newVehicleFixture(t *testing.T) *vehicleFixture {
	t.Helper()
	db := setupTestDB(t)
	repo := repository.NewTaskListRepository(db)
	ws := &mockWsCtxStore{
		workspace: &models.Workspace{Base: models.Base{ID: fxWorkspaceID}},
		memberRole: map[uint]models.WorkspaceRole{
			fxOwner:       models.RoleOwner,
			fxManager:     models.RoleManager,
			fxEditor:      models.RoleEditor,
			fxEditorOther: models.RoleEditor,
		},
	}
	return &vehicleFixture{db: db, repo: repo, svc: NewTaskListService(repo, repository.NewTaskItemRepository(db), ws)}
}

// createAssigned cria um veículo do dono com dados do cliente e atribui o mecânico fxEditor.
func (f *vehicleFixture) createAssigned(t *testing.T) *models.TaskList {
	t.Helper()
	list, err := f.svc.CreateList(fxOwner, VehicleInput{
		Title:            "Gol",
		Plate:            "abc1d23",
		Customer:         "Maria",
		CustomerCPF:      ptr("123.456.789-09"),
		CustomerCEP:      ptr("01310-100"),
		CustomerWhatsApp: ptr("+55 (11) 98765-4321"),
	})
	require.NoError(t, err)
	require.NoError(t, f.db.Create(&models.ListAssignment{TaskListID: list.ID, UserID: fxEditor, AssignedBy: fxOwner}).Error)
	return list
}

func (f *vehicleFixture) reload(t *testing.T, id uint) *models.TaskList {
	t.Helper()
	got, err := f.repo.FindByID(id)
	require.NoError(t, err)
	return got
}

func ptr[T any](v T) *T { return &v }

func jsonKeys(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))
	return body
}

var customerDataKeys = []string{"customer_cpf", "customer_cep", "customer_whatsapp"}

func assertNoCustomerData(t *testing.T, v any) {
	t.Helper()
	body := jsonKeys(t, v)
	for _, key := range customerDataKeys {
		assert.NotContains(t, body, key)
	}
	assert.Contains(t, body, "customer", "o nome do cliente continua visível")
}

func TestTaskListService_CreateList_StoresNormalizedCustomerData(t *testing.T) {
	f := newVehicleFixture(t)
	list := f.createAssigned(t)

	assert.True(t, list.CustomerDataVisible)
	got := f.reload(t, list.ID)
	assert.Equal(t, "12345678909", got.CustomerCPF)
	assert.Equal(t, "01310100", got.CustomerCEP)
	assert.Equal(t, "11987654321", got.CustomerWhatsApp)

	plain, err := f.svc.CreateList(fxManager, VehicleInput{Title: "Uno"})
	require.NoError(t, err)
	got = f.reload(t, plain.ID)
	assert.Equal(t, "", got.CustomerCPF)
	assert.Equal(t, "", got.CustomerCEP)
	assert.Equal(t, "", got.CustomerWhatsApp)
}

func TestTaskListService_CreateList_InvalidCustomerData(t *testing.T) {
	cases := []struct {
		in   VehicleInput
		want error
	}{
		{VehicleInput{Title: "Gol", CustomerCPF: ptr("111.111.111-11")}, ErrInvalidCPF},
		{VehicleInput{Title: "Gol", CustomerCEP: ptr("1234-567")}, ErrInvalidCEP},
		{VehicleInput{Title: "Gol", CustomerWhatsApp: ptr("98765-4321")}, ErrInvalidWhatsApp},
	}
	for _, tc := range cases {
		created := false
		store := &mockTaskListStore{createFunc: func(list *models.TaskList) error { created = true; return nil }}
		svc := NewTaskListService(store, &mockTaskItemStore{}, nil)

		_, err := svc.CreateList(1, tc.in)
		assert.ErrorIs(t, err, tc.want)
		assert.False(t, created)
	}
}

func TestTaskListService_UpdateList_AppStylePutKeepsCustomerData(t *testing.T) {
	f := newVehicleFixture(t)
	list := f.createAssigned(t)

	// O app só conhece title/plate/customer: nada de CPF/CEP/WhatsApp no body.
	updated, err := f.svc.UpdateList(list.ID, fxManager, VehicleInput{Title: "Gol G5", Plate: "abc1d23", Customer: "Maria Souza"})
	require.NoError(t, err)
	assert.True(t, updated.CustomerDataVisible)

	got := f.reload(t, list.ID)
	assert.Equal(t, "Gol G5", got.Title)
	assert.Equal(t, "Maria Souza", got.Customer)
	assert.Equal(t, "12345678909", got.CustomerCPF)
	assert.Equal(t, "01310100", got.CustomerCEP)
	assert.Equal(t, "11987654321", got.CustomerWhatsApp)
}

func TestTaskListService_UpdateList_PresentKeysSetOrClear(t *testing.T) {
	f := newVehicleFixture(t)
	list := f.createAssigned(t)

	_, err := f.svc.UpdateList(list.ID, fxOwner, VehicleInput{
		Title:            "Gol",
		CustomerCPF:      ptr(""),
		CustomerWhatsApp: ptr("(21) 3456-7890"),
	})
	require.NoError(t, err)

	got := f.reload(t, list.ID)
	assert.Equal(t, "", got.CustomerCPF, "chave presente com \"\" limpa")
	assert.Equal(t, "01310100", got.CustomerCEP, "chave ausente mantém")
	assert.Equal(t, "2134567890", got.CustomerWhatsApp)
}

func TestTaskListService_UpdateList_InvalidCustomerDataDoesNotPersist(t *testing.T) {
	f := newVehicleFixture(t)
	list := f.createAssigned(t)

	_, err := f.svc.UpdateList(list.ID, fxOwner, VehicleInput{
		Title:       "Outro título",
		CustomerCEP: ptr("99999-999"),
		CustomerCPF: ptr("123.456.789-00"),
	})
	assert.ErrorIs(t, err, ErrInvalidCPF)

	got := f.reload(t, list.ID)
	assert.Equal(t, "Gol", got.Title)
	assert.Equal(t, "12345678909", got.CustomerCPF)
	assert.Equal(t, "01310100", got.CustomerCEP)
}

func TestTaskListService_UpdateList_EditorForbidden(t *testing.T) {
	f := newVehicleFixture(t)
	list := f.createAssigned(t)

	_, err := f.svc.UpdateList(list.ID, fxEditor, VehicleInput{Title: "Gol", CustomerCPF: ptr("")})
	assert.ErrorIs(t, err, ErrNotManager)
	assert.Equal(t, "12345678909", f.reload(t, list.ID).CustomerCPF)
}

func TestTaskListService_CustomerDataVisibleOnlyToOwnerAndManager(t *testing.T) {
	f := newVehicleFixture(t)
	list := f.createAssigned(t)

	for _, userID := range []uint{fxOwner, fxManager} {
		detail, err := f.svc.GetList(list.ID, userID)
		require.NoError(t, err)
		body := jsonKeys(t, detail)
		assert.Equal(t, "12345678909", body["customer_cpf"])
		assert.Equal(t, "01310100", body["customer_cep"])
		assert.Equal(t, "11987654321", body["customer_whatsapp"])

		page, err := f.svc.ListAll(userID, "", 1, 20, "", false)
		require.NoError(t, err)
		require.Len(t, page.Lists, 1)
		assert.True(t, page.Lists[0].CustomerDataVisible)
	}

	detail, err := f.svc.GetList(list.ID, fxEditor)
	require.NoError(t, err)
	assertNoCustomerData(t, detail)

	for _, mine := range []bool{false, true} {
		page, err := f.svc.ListAll(fxEditor, "", 1, 20, "", mine)
		require.NoError(t, err)
		require.Len(t, page.Lists, 1)
		assertNoCustomerData(t, page.Lists[0])
		raw := string(mustJSON(t, page))
		for _, key := range customerDataKeys {
			assert.NotContains(t, raw, key)
		}
	}

	_, err = f.svc.GetList(list.ID, fxEditorOther)
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = f.svc.GetList(list.ID, fxOutsider)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestTaskListService_ListAll_StaleAssignmentWithoutWorkspaceHidesCustomerData(t *testing.T) {
	f := newVehicleFixture(t)
	list := f.createAssigned(t)
	// Ex-membro que ainda tem atribuição antiga e pede a esteira sem pertencer a oficina nenhuma.
	require.NoError(t, f.db.Create(&models.ListAssignment{TaskListID: list.ID, UserID: fxOutsider, AssignedBy: fxOwner}).Error)

	page, err := f.svc.ListAll(fxOutsider, "", 1, 20, "", true)
	require.NoError(t, err)
	require.Len(t, page.Lists, 1)
	assertNoCustomerData(t, page.Lists[0])
}

func TestViewer_CanSeeCustomerData(t *testing.T) {
	ws, other := fxWorkspaceID, fxOtherWorkspc
	inWS := &models.TaskList{UserID: fxOwner, WorkspaceID: &ws}
	inOther := &models.TaskList{UserID: fxOwner, WorkspaceID: &other}
	personalOwn := &models.TaskList{UserID: fxManager}
	personalOther := &models.TaskList{UserID: fxOwner}

	owner := viewer{userID: fxOwner, wsID: &ws, role: models.RoleOwner}
	manager := viewer{userID: fxManager, wsID: &ws, role: models.RoleManager}
	editor := viewer{userID: fxEditor, wsID: &ws, role: models.RoleEditor}
	noWorkspace := viewer{userID: fxManager}

	assert.True(t, owner.canSeeCustomerData(inWS))
	assert.True(t, manager.canSeeCustomerData(inWS))
	assert.False(t, editor.canSeeCustomerData(inWS))
	assert.False(t, owner.canSeeCustomerData(inOther))
	assert.True(t, manager.canSeeCustomerData(personalOwn))
	assert.False(t, manager.canSeeCustomerData(personalOther), "veículo legado de outro usuário")
	assert.False(t, editor.canSeeCustomerData(&models.TaskList{UserID: fxEditor}), "mecânico nunca recebe")
	assert.True(t, noWorkspace.canSeeCustomerData(personalOwn))
	assert.False(t, noWorkspace.canSeeCustomerData(inWS))
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return raw
}
