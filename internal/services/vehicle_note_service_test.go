package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
)

type noteFixture struct {
	db             *gorm.DB
	svc            *VehicleNoteService
	notes          *repository.VehicleNoteRepository
	lists          *repository.TaskListRepository
	owner          uint
	manager        uint
	editorAssigned uint
	editorOther    uint
	outsider       uint
	vehicle        uint
	otherVehicle   uint
	foreignVehicle uint
}

func newNoteFixture(t *testing.T) *noteFixture {
	t.Helper()
	db := setupTestDB(t)

	mkUser := func(username, first, last string) uint {
		u := &models.User{Username: username, FirstName: first, LastName: last, Password: "hash"}
		require.NoError(t, db.Create(u).Error)
		return u.ID
	}
	f := &noteFixture{db: db}
	f.owner = mkUser("dono@oficina.com", "Dora", "Dona")
	f.manager = mkUser("gerente@oficina.com", "Gil", "Gerente")
	f.editorAssigned = mkUser("mecanico@oficina.com", "Mário", "Mecânico")
	f.editorOther = mkUser("outro@oficina.com", "", "")
	f.outsider = mkUser("fora@outra.com", "Fábio", "Fora")

	ws := &models.Workspace{Name: "RMAXX", OwnerID: f.owner}
	require.NoError(t, db.Create(ws).Error)
	otherWs := &models.Workspace{Name: "Outra", OwnerID: f.outsider}
	require.NoError(t, db.Create(otherWs).Error)
	for _, m := range []models.WorkspaceMember{
		{WorkspaceID: ws.ID, UserID: f.owner, Role: models.RoleOwner},
		{WorkspaceID: ws.ID, UserID: f.manager, Role: models.RoleManager},
		{WorkspaceID: ws.ID, UserID: f.editorAssigned, Role: models.RoleEditor},
		{WorkspaceID: ws.ID, UserID: f.editorOther, Role: models.RoleEditor},
		{WorkspaceID: otherWs.ID, UserID: f.outsider, Role: models.RoleOwner},
	} {
		m := m
		require.NoError(t, db.Create(&m).Error)
	}

	f.lists = repository.NewTaskListRepository(db)
	mkVehicle := func(userID, wsID uint) uint {
		id := wsID
		list := &models.TaskList{Title: "Veículo", UserID: userID, WorkspaceID: &id, Status: models.StatusEmAndamento}
		require.NoError(t, f.lists.Create(list))
		return list.ID
	}
	f.vehicle = mkVehicle(f.owner, ws.ID)
	f.otherVehicle = mkVehicle(f.owner, ws.ID)
	f.foreignVehicle = mkVehicle(f.outsider, otherWs.ID)
	require.NoError(t, db.Create(&models.ListAssignment{TaskListID: f.vehicle, UserID: f.editorAssigned, AssignedBy: f.owner}).Error)

	f.notes = repository.NewVehicleNoteRepository(db)
	f.svc = NewVehicleNoteService(f.notes, f.lists, repository.NewWorkspaceRepository(db))
	return f
}

func (f *noteFixture) add(t *testing.T, userID uint, text string) *models.VehicleNote {
	t.Helper()
	n, err := f.svc.AddNote(f.vehicle, userID, text)
	require.NoError(t, err)
	return n
}

func TestVehicleNoteService_AddNote_TrimsAndFillsAuthor(t *testing.T) {
	f := newNoteFixture(t)

	n, err := f.svc.AddNote(f.vehicle, f.owner, "  Cliente avisou ruído\nna suspensão  ")

	require.NoError(t, err)
	assert.Equal(t, "Cliente avisou ruído\nna suspensão", n.Text)
	assert.Equal(t, f.vehicle, n.TaskListID)
	assert.Equal(t, f.owner, n.AuthorID)
	assert.Equal(t, "Dora Dona", n.AuthorName)
}

func TestVehicleNoteService_AddNote_AuthorNameFallsBackToUsername(t *testing.T) {
	f := newNoteFixture(t)
	require.NoError(t, f.db.Create(&models.ListAssignment{TaskListID: f.vehicle, UserID: f.editorOther, AssignedBy: f.owner}).Error)

	n := f.add(t, f.editorOther, "Sem nome cadastrado")

	assert.Equal(t, "outro@oficina.com", n.AuthorName)
}

func TestVehicleNoteService_AddNote_Validation(t *testing.T) {
	f := newNoteFixture(t)

	_, err := f.svc.AddNote(f.vehicle, f.owner, "")
	assert.ErrorIs(t, err, ErrNoteTextRequired)

	_, err = f.svc.AddNote(f.vehicle, f.owner, " \n\t ")
	assert.ErrorIs(t, err, ErrNoteTextRequired)

	_, err = f.svc.AddNote(f.vehicle, f.owner, strings.Repeat("é", MaxVehicleNoteLength+1))
	assert.ErrorIs(t, err, ErrNoteTextTooLong)

	n, err := f.svc.AddNote(f.vehicle, f.owner, strings.Repeat("é", MaxVehicleNoteLength))
	require.NoError(t, err)
	assert.Len(t, []rune(n.Text), MaxVehicleNoteLength)
}

func TestVehicleNoteService_AccessByRole(t *testing.T) {
	f := newNoteFixture(t)

	for _, userID := range []uint{f.owner, f.manager, f.editorAssigned} {
		_, err := f.svc.AddNote(f.vehicle, userID, "ok")
		assert.NoError(t, err)
		_, err = f.svc.ListNotes(f.vehicle, userID)
		assert.NoError(t, err)
	}

	for _, userID := range []uint{f.editorOther, f.outsider} {
		_, err := f.svc.AddNote(f.vehicle, userID, "negado")
		assert.ErrorIs(t, err, ErrNotFound)
		_, err = f.svc.ListNotes(f.vehicle, userID)
		assert.ErrorIs(t, err, ErrNotFound)
	}

	_, err := f.svc.ListNotes(f.foreignVehicle, f.owner)
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = f.svc.ListNotes(9999, f.owner)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestVehicleNoteService_AccessCheckedBeforeValidation(t *testing.T) {
	f := newNoteFixture(t)

	_, err := f.svc.AddNote(f.foreignVehicle, f.owner, "")

	assert.ErrorIs(t, err, ErrNotFound)
}

func TestVehicleNoteService_ListNotes_NewestFirstWithoutDeleted(t *testing.T) {
	f := newNoteFixture(t)
	first := f.add(t, f.owner, "primeira")
	second := f.add(t, f.editorAssigned, "segunda")
	third := f.add(t, f.manager, "terceira")
	require.NoError(t, f.svc.DeleteNote(f.vehicle, second.ID, f.manager))

	notes, err := f.svc.ListNotes(f.vehicle, f.editorAssigned)

	require.NoError(t, err)
	require.Len(t, notes, 2)
	assert.Equal(t, third.ID, notes[0].ID)
	assert.Equal(t, first.ID, notes[1].ID)
}

func TestVehicleNoteService_DeletedNoteHiddenFromVehicle(t *testing.T) {
	f := newNoteFixture(t)
	kept := f.add(t, f.owner, "fica")
	removed := f.add(t, f.editorAssigned, "sai")
	require.NoError(t, f.svc.DeleteNote(f.vehicle, removed.ID, f.editorAssigned))

	lists := NewTaskListService(f.lists, repository.NewTaskItemRepository(f.db), repository.NewWorkspaceRepository(f.db))
	vehicle, err := lists.GetList(f.vehicle, f.editorAssigned)
	require.NoError(t, err)
	require.Len(t, vehicle.Notes, 1)
	assert.Equal(t, kept.ID, vehicle.Notes[0].ID)

	page, err := lists.ListAll(f.owner, "", 1, 20, "", false)
	require.NoError(t, err)
	for _, v := range page.Lists {
		for _, n := range v.Notes {
			assert.NotEqual(t, removed.ID, n.ID)
		}
	}
}

func TestVehicleNoteService_DeleteNote_ManagersDeleteAny(t *testing.T) {
	f := newNoteFixture(t)

	for _, userID := range []uint{f.owner, f.manager} {
		n := f.add(t, f.editorAssigned, "do mecânico")
		require.NoError(t, f.svc.DeleteNote(f.vehicle, n.ID, userID))

		var raw models.VehicleNote
		require.NoError(t, f.db.Unscoped().First(&raw, n.ID).Error)
		assert.True(t, raw.DeletedAt.Valid)
		require.NotNil(t, raw.DeletedBy)
		assert.Equal(t, userID, *raw.DeletedBy)
	}
}

func TestVehicleNoteService_DeleteNote_AuthorDeletesOwn(t *testing.T) {
	f := newNoteFixture(t)
	n := f.add(t, f.editorAssigned, "minha")

	require.NoError(t, f.svc.DeleteNote(f.vehicle, n.ID, f.editorAssigned))

	notes, err := f.svc.ListNotes(f.vehicle, f.owner)
	require.NoError(t, err)
	assert.Empty(t, notes)
}

func TestVehicleNoteService_DeleteNote_EditorCannotDeleteOthers(t *testing.T) {
	f := newNoteFixture(t)
	n := f.add(t, f.owner, "do dono")

	err := f.svc.DeleteNote(f.vehicle, n.ID, f.editorAssigned)

	assert.ErrorIs(t, err, ErrNoteForbidden)
	notes, _ := f.svc.ListNotes(f.vehicle, f.owner)
	assert.Len(t, notes, 1)
}

func TestVehicleNoteService_DeleteNote_NoAccessReturnsNotFound(t *testing.T) {
	f := newNoteFixture(t)
	n := f.add(t, f.owner, "x")

	assert.ErrorIs(t, f.svc.DeleteNote(f.vehicle, n.ID, f.editorOther), ErrNotFound)
	assert.ErrorIs(t, f.svc.DeleteNote(f.vehicle, n.ID, f.outsider), ErrNotFound)
}

func TestVehicleNoteService_DeleteNote_AlreadyDeletedOrOtherVehicle(t *testing.T) {
	f := newNoteFixture(t)
	n := f.add(t, f.owner, "x")
	require.NoError(t, f.svc.DeleteNote(f.vehicle, n.ID, f.owner))

	assert.ErrorIs(t, f.svc.DeleteNote(f.vehicle, n.ID, f.owner), ErrNoteNotFound)

	other := f.add(t, f.owner, "y")
	assert.ErrorIs(t, f.svc.DeleteNote(f.otherVehicle, other.ID, f.owner), ErrNoteNotFound)
	assert.ErrorIs(t, f.svc.DeleteNote(f.vehicle, 9999, f.owner), ErrNoteNotFound)
}

type mockNoteStore struct {
	createErr  error
	note       *models.VehicleNote
	softDelErr error
}

func (m *mockNoteStore) Create(n *models.VehicleNote) error { return m.createErr }
func (m *mockNoteStore) FindByID(id uint) (*models.VehicleNote, error) {
	if m.note == nil {
		return nil, errors.New("not found")
	}
	return m.note, nil
}
func (m *mockNoteStore) ListByTaskList(taskListID uint) ([]models.VehicleNote, error) {
	return nil, nil
}
func (m *mockNoteStore) SoftDelete(n *models.VehicleNote, deletedBy uint) error { return m.softDelErr }

func TestVehicleNoteService_StoreErrorsPropagate(t *testing.T) {
	dbErr := errors.New("db down")

	svc := NewVehicleNoteService(&mockNoteStore{createErr: dbErr}, quoteListStore(1), nil)
	_, err := svc.AddNote(1, 1, "texto")
	assert.ErrorIs(t, err, dbErr)

	note := &models.VehicleNote{ID: 3, TaskListID: 1, AuthorID: 1}
	svc = NewVehicleNoteService(&mockNoteStore{note: note, softDelErr: dbErr}, quoteListStore(1), nil)
	assert.ErrorIs(t, svc.DeleteNote(1, 3, 1), dbErr)

	svc = NewVehicleNoteService(&mockNoteStore{note: note, softDelErr: gorm.ErrRecordNotFound}, quoteListStore(1), nil)
	assert.ErrorIs(t, svc.DeleteNote(1, 3, 1), ErrNoteNotFound)
}

func TestVehicleNoteService_DeleteNote_PersonalVehicleCreatorIsOwner(t *testing.T) {
	note := &models.VehicleNote{ID: 3, TaskListID: 1, AuthorID: 2}
	svc := NewVehicleNoteService(&mockNoteStore{note: note}, quoteListStore(1), nil)

	assert.NoError(t, svc.DeleteNote(1, 3, 1))
}

func TestVehicleNoteService_DeleteNote_RoleLookupErrorForbids(t *testing.T) {
	wsID := uint(10)
	lists := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return &models.TaskList{Base: models.Base{ID: id}, UserID: 5, WorkspaceID: &wsID}, nil
		},
	}
	note := &models.VehicleNote{ID: 3, TaskListID: 1, AuthorID: 2}
	svc := NewVehicleNoteService(&mockNoteStore{note: note}, lists, &mockAccessStore{roleErr: errors.New("sem papel")})

	assert.ErrorIs(t, svc.DeleteNote(1, 3, 5), ErrNoteForbidden)
}
