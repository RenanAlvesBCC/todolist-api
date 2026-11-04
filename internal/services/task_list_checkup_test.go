package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func (f *vehicleFixture) createUsers(t *testing.T) {
	t.Helper()
	users := []models.User{
		{Base: models.Base{ID: fxOwner}, Username: "dono@oficina.com", FirstName: "Rafael", LastName: "Max"},
		{Base: models.Base{ID: fxManager}, Username: "gerente@oficina.com", FirstName: " Bia ", LastName: ""},
		{Base: models.Base{ID: fxEditor}, Username: "mecanico@oficina.com"},
		{Base: models.Base{ID: fxEditorOther}, Username: "outro@oficina.com"},
	}
	for i := range users {
		require.NoError(t, f.db.Create(&users[i]).Error)
	}
}

func TestTaskListService_SetCheckupVideo_SetKeepClear(t *testing.T) {
	f := newVehicleFixture(t)
	f.createUsers(t)
	list := f.createAssigned(t)
	assert.Nil(t, f.reload(t, list.ID).CheckupVideoDoneAt, "veículo novo começa sem vídeo")

	before := time.Now().UTC().Add(-time.Second)
	done, err := f.svc.SetCheckupVideo(list.ID, fxOwner, true)
	require.NoError(t, err)
	require.NotNil(t, done.CheckupVideoDoneAt)
	require.NotNil(t, done.CheckupVideoDoneBy)
	require.NotNil(t, done.CheckupVideoDoneByName)
	assert.False(t, done.CheckupVideoDoneAt.Before(before))
	assert.Equal(t, fxOwner, *done.CheckupVideoDoneBy)
	assert.Equal(t, "Rafael Max", *done.CheckupVideoDoneByName)
	assert.True(t, done.CustomerDataVisible)
	firstAt := *done.CheckupVideoDoneAt

	again, err := f.svc.SetCheckupVideo(list.ID, fxManager, true)
	require.NoError(t, err)
	assert.True(t, firstAt.Equal(*again.CheckupVideoDoneAt), "reenviar true mantém a data original")
	assert.Equal(t, fxOwner, *again.CheckupVideoDoneBy, "e quem marcou primeiro")
	stored := f.reload(t, list.ID)
	assert.True(t, firstAt.Equal(*stored.CheckupVideoDoneAt))

	cleared, err := f.svc.SetCheckupVideo(list.ID, fxManager, false)
	require.NoError(t, err)
	assert.Nil(t, cleared.CheckupVideoDoneAt)
	assert.Nil(t, cleared.CheckupVideoDoneBy)
	assert.Nil(t, cleared.CheckupVideoDoneByName)
	stored = f.reload(t, list.ID)
	assert.Nil(t, stored.CheckupVideoDoneAt)
	assert.Nil(t, stored.CheckupVideoDoneBy)

	_, err = f.svc.SetCheckupVideo(list.ID, fxManager, false)
	require.NoError(t, err, "desmarcar de novo é idempotente")

	byManager, err := f.svc.SetCheckupVideo(list.ID, fxManager, true)
	require.NoError(t, err)
	assert.Equal(t, "Bia", *byManager.CheckupVideoDoneByName, "nome com trim")
}

func TestTaskListService_SetCheckupVideo_EditorAssignedCanMark(t *testing.T) {
	f := newVehicleFixture(t)
	f.createUsers(t)
	list := f.createAssigned(t)

	done, err := f.svc.SetCheckupVideo(list.ID, fxEditor, true)
	require.NoError(t, err)
	assert.Equal(t, fxEditor, *done.CheckupVideoDoneBy)
	assert.Equal(t, "mecanico@oficina.com", *done.CheckupVideoDoneByName, "sem nome cai no username")
	assert.False(t, done.CustomerDataVisible)
	assertNoCustomerData(t, done)

	body := jsonKeys(t, done)
	assert.NotNil(t, body["checkup_video_done_at"])
	assert.Equal(t, float64(fxEditor), body["checkup_video_done_by"])
	assert.Equal(t, "mecanico@oficina.com", body["checkup_video_done_by_name"])

	cleared, err := f.svc.SetCheckupVideo(list.ID, fxEditor, false)
	require.NoError(t, err)
	assert.Nil(t, cleared.CheckupVideoDoneAt)
}

func TestTaskListService_SetCheckupVideo_NoAccessReturnsNotFound(t *testing.T) {
	f := newVehicleFixture(t)
	f.createUsers(t)
	list := f.createAssigned(t)

	for name, userID := range map[string]uint{
		"mecânico não atribuído": fxEditorOther,
		"outra oficina":          fxOutsider,
	} {
		_, err := f.svc.SetCheckupVideo(list.ID, userID, true)
		assert.ErrorIs(t, err, ErrNotFound, name)
	}
	_, err := f.svc.SetCheckupVideo(9999, fxOwner, true)
	assert.ErrorIs(t, err, ErrNotFound)

	assert.Nil(t, f.reload(t, list.ID).CheckupVideoDoneAt)
}

func TestTaskListService_SetCheckupVideo_VisibleInListAndDetail(t *testing.T) {
	f := newVehicleFixture(t)
	f.createUsers(t)
	list := f.createAssigned(t)
	_, err := f.svc.SetCheckupVideo(list.ID, fxManager, true)
	require.NoError(t, err)

	for _, userID := range []uint{fxOwner, fxEditor} {
		page, err := f.svc.ListAll(userID, "", 1, 20, "", false)
		require.NoError(t, err)
		require.Len(t, page.Lists, 1)
		require.NotNil(t, page.Lists[0].CheckupVideoDoneByName)
		assert.Equal(t, "Bia", *page.Lists[0].CheckupVideoDoneByName)

		detail, err := f.svc.GetList(list.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, fxManager, *detail.CheckupVideoDoneBy)
	}
}

func TestTaskListService_SetCheckupVideo_StoreErrors(t *testing.T) {
	current := func() *models.TaskList { return &models.TaskList{Base: models.Base{ID: 1}, UserID: 1} }

	svc := NewTaskListService(&mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) { return current(), nil },
		updateFunc:   func(list *models.TaskList) error { return assert.AnError },
	}, &mockTaskItemStore{}, nil)
	_, err := svc.SetCheckupVideo(1, 1, true)
	assert.ErrorIs(t, err, assert.AnError)

	calls := 0
	svc = NewTaskListService(&mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			calls++
			if calls > 1 {
				return nil, assert.AnError
			}
			return current(), nil
		},
	}, &mockTaskItemStore{}, nil)
	_, err = svc.SetCheckupVideo(1, 1, true)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestApplyCheckupVideo(t *testing.T) {
	now := time.Date(2026, 11, 4, 14, 0, 0, 0, time.UTC)
	earlier := now.Add(-time.Hour)
	by := uint(2)
	name := "Ana"

	list := &models.TaskList{}
	assert.True(t, applyCheckupVideo(list, true, 5, now))
	assert.True(t, now.Equal(*list.CheckupVideoDoneAt))
	assert.Equal(t, uint(5), *list.CheckupVideoDoneBy)

	list = &models.TaskList{CheckupVideoDoneAt: &earlier, CheckupVideoDoneBy: &by}
	assert.False(t, applyCheckupVideo(list, true, 5, now))
	assert.True(t, earlier.Equal(*list.CheckupVideoDoneAt))
	assert.Equal(t, by, *list.CheckupVideoDoneBy)

	list = &models.TaskList{CheckupVideoDoneAt: &earlier, CheckupVideoDoneBy: &by, CheckupVideoDoneByName: &name}
	assert.True(t, applyCheckupVideo(list, false, 5, now))
	assert.Nil(t, list.CheckupVideoDoneAt)
	assert.Nil(t, list.CheckupVideoDoneBy)
	assert.Nil(t, list.CheckupVideoDoneByName)

	assert.False(t, applyCheckupVideo(&models.TaskList{}, false, 5, now))
}
