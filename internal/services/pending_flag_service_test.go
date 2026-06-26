package services

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

type mockFlagStore struct {
	created  []models.PendingFlag
	findByID func(id uint) (*models.PendingFlag, error)
	list     []models.PendingFlag
	updated  *models.PendingFlag
}

func (m *mockFlagStore) Create(f *models.PendingFlag) error {
	f.ID = uint(len(m.created) + 1)
	m.created = append(m.created, *f)
	return nil
}
func (m *mockFlagStore) FindByID(id uint) (*models.PendingFlag, error) {
	if m.findByID != nil {
		return m.findByID(id)
	}
	return nil, errors.New("not found")
}
func (m *mockFlagStore) ListByTaskList(taskListID uint) ([]models.PendingFlag, error) {
	return m.list, nil
}
func (m *mockFlagStore) Update(f *models.PendingFlag) error {
	m.updated = f
	return nil
}

func flagListStore(userID uint) *mockTaskListStore {
	return &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return &models.TaskList{Model: gorm.Model{ID: id}, UserID: userID}, nil
		},
	}
}

func TestPendingFlagService_AddFlag_Success(t *testing.T) {
	store := &mockFlagStore{}
	svc := NewPendingFlagService(store, flagListStore(1), nil)

	f, err := svc.AddFlag(1, 1, models.FlagAguardandoCliente, "")

	require.NoError(t, err)
	assert.Equal(t, models.FlagAguardandoCliente, f.FlagType)
	assert.Len(t, store.created, 1)
}

func TestPendingFlagService_AddFlag_OutroRequiresNote(t *testing.T) {
	svc := NewPendingFlagService(&mockFlagStore{}, flagListStore(1), nil)

	_, err := svc.AddFlag(1, 1, models.FlagOutro, "curto")

	assert.EqualError(t, err, "nota obrigatória com mínimo de 10 caracteres para tipo 'outro'")
}

func TestPendingFlagService_AddFlag_OutroWithValidNote(t *testing.T) {
	store := &mockFlagStore{}
	svc := NewPendingFlagService(store, flagListStore(1), nil)

	_, err := svc.AddFlag(1, 1, models.FlagOutro, "Aguardando peça importada")

	require.NoError(t, err)
}

func TestPendingFlagService_ResolveFlag_Success(t *testing.T) {
	flag := &models.PendingFlag{ID: 3, TaskListID: 1}
	store := &mockFlagStore{
		findByID: func(id uint) (*models.PendingFlag, error) { return flag, nil },
	}
	svc := NewPendingFlagService(store, flagListStore(1), nil)

	err := svc.ResolveFlag(1, 3, 1)

	require.NoError(t, err)
	require.NotNil(t, store.updated.ResolvedAt)
	assert.Equal(t, uint(1), *store.updated.ResolvedBy)
}

func TestPendingFlagService_ResolveFlag_AlreadyResolved(t *testing.T) {
	now := time.Now()
	flag := &models.PendingFlag{ID: 3, TaskListID: 1, ResolvedAt: &now}
	store := &mockFlagStore{
		findByID: func(id uint) (*models.PendingFlag, error) { return flag, nil },
	}
	svc := NewPendingFlagService(store, flagListStore(1), nil)

	err := svc.ResolveFlag(1, 3, 1)

	assert.EqualError(t, err, "pendência já resolvida")
}

func TestPendingFlagService_ResolveFlag_WrongListReturnsError(t *testing.T) {
	store := &mockFlagStore{
		findByID: func(id uint) (*models.PendingFlag, error) {
			return &models.PendingFlag{ID: id, TaskListID: 99}, nil
		},
	}
	svc := NewPendingFlagService(store, flagListStore(1), nil)

	err := svc.ResolveFlag(1, 3, 1)

	assert.EqualError(t, err, "pendência não encontrada")
}

func TestPendingFlagService_ListFlags_ReturnsAll(t *testing.T) {
	now := time.Now()
	store := &mockFlagStore{
		list: []models.PendingFlag{
			{ID: 1, FlagType: models.FlagAguardandoCliente},
			{ID: 2, FlagType: models.FlagOutro, ResolvedAt: &now},
		},
	}
	svc := NewPendingFlagService(store, flagListStore(1), nil)

	flags, err := svc.ListFlags(1, 1)

	require.NoError(t, err)
	assert.Len(t, flags, 2)
}
