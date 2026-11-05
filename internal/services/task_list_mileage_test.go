package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func TestTaskListService_CreateList_Mileage(t *testing.T) {
	f := newVehicleFixture(t)

	plain, err := f.svc.CreateList(fxOwner, VehicleInput{Title: "Gol"})
	require.NoError(t, err)
	assert.Nil(t, plain.Mileage)
	assert.Nil(t, f.reload(t, plain.ID).Mileage, "sem mileage fica NULL")

	for _, km := range []int{0, 123456, MaxMileage} {
		list, err := f.svc.CreateList(fxManager, VehicleInput{Title: "Uno", Mileage: ptr(km)})
		require.NoError(t, err)
		got := f.reload(t, list.ID).Mileage
		require.NotNil(t, got)
		assert.Equal(t, km, *got)
	}
}

func TestTaskListService_CreateList_InvalidMileage(t *testing.T) {
	for _, km := range []int{-1, MaxMileage + 1} {
		created := false
		store := &mockTaskListStore{createFunc: func(list *models.TaskList) error { created = true; return nil }}
		svc := NewTaskListService(store, &mockTaskItemStore{}, nil)

		_, err := svc.CreateList(1, VehicleInput{Title: "Gol", Mileage: ptr(km)})
		assert.ErrorIs(t, err, ErrInvalidMileage)
		assert.Equal(t, "quilometragem inválida", err.Error())
		assert.False(t, created)
	}
}

func TestTaskListService_UpdateList_MileageAbsentKeepsPresentSets(t *testing.T) {
	f := newVehicleFixture(t)
	list := f.createAssigned(t)
	_, err := f.svc.UpdateList(list.ID, fxOwner, VehicleInput{Title: "Gol", Plate: "abc1d23", Customer: "Maria", Priority: ptr("alta"), Mileage: ptr(87000)})
	require.NoError(t, err)

	// O app só conhece title/plate/customer: quilometragem e demais opcionais ficam.
	_, err = f.svc.UpdateList(list.ID, fxManager, VehicleInput{Title: "Gol G5", Plate: "abc1d23", Customer: "Maria Souza"})
	require.NoError(t, err)
	got := f.reload(t, list.ID)
	assert.Equal(t, "Gol G5", got.Title)
	require.NotNil(t, got.Mileage)
	assert.Equal(t, 87000, *got.Mileage)
	assert.Equal(t, models.PriorityAlta, got.Priority)
	assert.Equal(t, "12345678909", got.CustomerCPF)
	assert.Equal(t, "01310100", got.CustomerCEP)
	assert.Equal(t, "11987654321", got.CustomerWhatsApp)

	updated, err := f.svc.UpdateList(list.ID, fxManager, VehicleInput{Title: "Gol G5", Mileage: ptr(0)})
	require.NoError(t, err)
	require.NotNil(t, updated.Mileage)
	assert.Equal(t, 0, *updated.Mileage)
	require.NotNil(t, f.reload(t, list.ID).Mileage)
	assert.Equal(t, 0, *f.reload(t, list.ID).Mileage, "0 é quilometragem válida, não limpa")
}

func TestTaskListService_UpdateList_InvalidMileageDoesNotPersist(t *testing.T) {
	f := newVehicleFixture(t)
	list, err := f.svc.CreateList(fxOwner, VehicleInput{Title: "Gol", Mileage: ptr(5000)})
	require.NoError(t, err)

	for _, km := range []int{-10, MaxMileage + 1} {
		_, err = f.svc.UpdateList(list.ID, fxOwner, VehicleInput{Title: "Outro", Priority: ptr("urgente"), Mileage: ptr(km)})
		assert.ErrorIs(t, err, ErrInvalidMileage)
	}
	_, err = f.svc.UpdateList(list.ID, fxOwner, VehicleInput{Title: "Outro", Mileage: ptr(9000), CustomerCPF: ptr("1")})
	assert.ErrorIs(t, err, ErrInvalidCPF)

	got := f.reload(t, list.ID)
	assert.Equal(t, "Gol", got.Title)
	assert.Equal(t, models.PriorityNormal, got.Priority)
	require.NotNil(t, got.Mileage)
	assert.Equal(t, 5000, *got.Mileage, "nada é gravado se algum campo for inválido")
}

func TestTaskListService_UpdateList_EditorCannotSetMileage(t *testing.T) {
	f := newVehicleFixture(t)
	list := f.createAssigned(t)

	_, err := f.svc.UpdateList(list.ID, fxEditor, VehicleInput{Title: "Gol", Mileage: ptr(1)})
	assert.ErrorIs(t, err, ErrNotManager)
	assert.Nil(t, f.reload(t, list.ID).Mileage)
}

func TestTaskListService_MileageVisibleToAllRoles(t *testing.T) {
	f := newVehicleFixture(t)
	list := f.createAssigned(t)
	_, err := f.svc.UpdateList(list.ID, fxOwner, VehicleInput{Title: "Gol", Mileage: ptr(42000)})
	require.NoError(t, err)

	for _, userID := range []uint{fxOwner, fxManager, fxEditor} {
		detail, err := f.svc.GetList(list.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, float64(42000), jsonKeys(t, detail)["mileage"])

		page, err := f.svc.ListAll(userID, "", 1, 20, "", false)
		require.NoError(t, err)
		require.Len(t, page.Lists, 1)
		assert.Equal(t, float64(42000), jsonKeys(t, page.Lists[0])["mileage"])
	}

	unknown, err := f.svc.CreateList(fxOwner, VehicleInput{Title: "Uno"})
	require.NoError(t, err)
	body := jsonKeys(t, unknown)
	assert.Contains(t, body, "mileage")
	assert.Nil(t, body["mileage"], "sem quilometragem vem null")
}
