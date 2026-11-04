package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func TestTaskListService_CreateList_Priority(t *testing.T) {
	f := newVehicleFixture(t)

	def, err := f.svc.CreateList(fxOwner, VehicleInput{Title: "Gol"})
	require.NoError(t, err)
	assert.Equal(t, models.PriorityNormal, def.Priority)
	assert.Equal(t, models.PriorityNormal, f.reload(t, def.ID).Priority)

	urgent, err := f.svc.CreateList(fxManager, VehicleInput{Title: "Uno", Priority: ptr("urgente")})
	require.NoError(t, err)
	assert.Equal(t, models.PriorityUrgente, f.reload(t, urgent.ID).Priority)

	_, err = f.svc.CreateList(fxOwner, VehicleInput{Title: "Fox", Priority: ptr("baixa")})
	assert.ErrorIs(t, err, ErrInvalidPriority)
	_, err = f.svc.CreateList(fxOwner, VehicleInput{Title: "Fox", Priority: ptr("")})
	assert.ErrorIs(t, err, ErrInvalidPriority)
}

func TestTaskListService_UpdateList_PriorityAbsentKeepsPresentSets(t *testing.T) {
	f := newVehicleFixture(t)
	list, err := f.svc.CreateList(fxOwner, VehicleInput{Title: "Gol", CustomerCPF: ptr("12345678909"), Priority: ptr("alta")})
	require.NoError(t, err)

	_, err = f.svc.UpdateList(list.ID, fxManager, VehicleInput{Title: "Gol G5", Plate: "ABC1D23", Customer: "Maria"})
	require.NoError(t, err)
	got := f.reload(t, list.ID)
	assert.Equal(t, models.PriorityAlta, got.Priority, "PUT do app (sem priority) mantém")
	assert.Equal(t, "12345678909", got.CustomerCPF)

	_, err = f.svc.UpdateList(list.ID, fxManager, VehicleInput{Title: "Gol G5", Priority: ptr("normal")})
	require.NoError(t, err)
	assert.Equal(t, models.PriorityNormal, f.reload(t, list.ID).Priority)

	_, err = f.svc.UpdateList(list.ID, fxManager, VehicleInput{Title: "Outro", Priority: ptr("máxima")})
	assert.ErrorIs(t, err, ErrInvalidPriority)
	assert.Equal(t, "Gol G5", f.reload(t, list.ID).Title)

	_, err = f.svc.UpdateList(list.ID, fxManager, VehicleInput{Title: "Outro", Priority: ptr("urgente"), CustomerCEP: ptr("1")})
	assert.ErrorIs(t, err, ErrInvalidCEP)
	assert.Equal(t, models.PriorityNormal, f.reload(t, list.ID).Priority, "nada é gravado se algum campo for inválido")
}

func TestTaskListService_ChangePriority(t *testing.T) {
	f := newVehicleFixture(t)
	list := f.createAssigned(t)

	for _, tc := range []struct {
		userID uint
		value  string
		want   models.VehiclePriority
	}{
		{fxOwner, "alta", models.PriorityAlta},
		{fxManager, "urgente", models.PriorityUrgente},
		{fxOwner, "normal", models.PriorityNormal},
	} {
		updated, err := f.svc.ChangePriority(list.ID, tc.userID, tc.value)
		require.NoError(t, err)
		assert.Equal(t, tc.want, updated.Priority)
		assert.True(t, updated.CustomerDataVisible)
		assert.Equal(t, tc.want, f.reload(t, list.ID).Priority)
	}

	positionBefore := f.reload(t, list.ID).Position
	_, err := f.svc.ChangePriority(list.ID, fxOwner, "urgente")
	require.NoError(t, err)
	assert.Equal(t, positionBefore, f.reload(t, list.ID).Position, "prioridade não reordena o pátio")

	_, err = f.svc.ChangePriority(list.ID, fxEditor, "alta")
	assert.ErrorIs(t, err, ErrNotManager, "mecânico atribuído não muda prioridade")
	_, err = f.svc.ChangePriority(list.ID, fxOwner, "baixa")
	assert.ErrorIs(t, err, ErrInvalidPriority)
	_, err = f.svc.ChangePriority(9999, fxOwner, "alta")
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = f.svc.ChangePriority(list.ID, fxOutsider, "alta")
	assert.ErrorIs(t, err, ErrNotFound, "outra oficina")
	assert.Equal(t, models.PriorityUrgente, f.reload(t, list.ID).Priority)
}

func TestTaskListService_PriorityVisibleToEditor(t *testing.T) {
	f := newVehicleFixture(t)
	list := f.createAssigned(t)
	_, err := f.svc.ChangePriority(list.ID, fxOwner, "urgente")
	require.NoError(t, err)

	detail, err := f.svc.GetList(list.ID, fxEditor)
	require.NoError(t, err)
	body := jsonKeys(t, detail)
	assert.Equal(t, "urgente", body["priority"])
	assertNoCustomerData(t, detail)
}
