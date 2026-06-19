package services

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
	"github.com/RenanAlvesBCC/todolist-api/internal/repository"
)

type mockTaskListStore struct {
	createFunc        func(list *models.TaskList) error
	findAllByUserFunc func(userID uint, filter repository.TaskListFilter) ([]models.TaskList, int64, error)
	findByIDFunc      func(id, userID uint) (*models.TaskList, error)
	updateFunc        func(list *models.TaskList) error
	deleteFunc        func(list *models.TaskList) error
}

func (m *mockTaskListStore) Create(list *models.TaskList) error { return m.createFunc(list) }
func (m *mockTaskListStore) FindAllByUser(userID uint, filter repository.TaskListFilter) ([]models.TaskList, int64, error) {
	return m.findAllByUserFunc(userID, filter)
}
func (m *mockTaskListStore) FindByIDAndUser(id, userID uint) (*models.TaskList, error) {
	return m.findByIDFunc(id, userID)
}
func (m *mockTaskListStore) Update(list *models.TaskList) error { return m.updateFunc(list) }
func (m *mockTaskListStore) Delete(list *models.TaskList) error { return m.deleteFunc(list) }

type mockTaskItemStore struct {
	createFunc          func(item *models.TaskItem) error
	findByIDAndListFunc func(id, taskListID uint) (*models.TaskItem, error)
	updateFunc          func(item *models.TaskItem) error
	deleteFunc          func(item *models.TaskItem) error
	deleteAllByListFunc func(taskListID uint) error
}

func (m *mockTaskItemStore) Create(item *models.TaskItem) error { return m.createFunc(item) }
func (m *mockTaskItemStore) FindByIDAndList(id, taskListID uint) (*models.TaskItem, error) {
	return m.findByIDAndListFunc(id, taskListID)
}
func (m *mockTaskItemStore) Update(item *models.TaskItem) error { return m.updateFunc(item) }
func (m *mockTaskItemStore) Delete(item *models.TaskItem) error { return m.deleteFunc(item) }
func (m *mockTaskItemStore) DeleteAllByList(taskListID uint) error {
	return m.deleteAllByListFunc(taskListID)
}

func TestTaskListService_CreateList_Success(t *testing.T) {
	store := &mockTaskListStore{
		createFunc: func(list *models.TaskList) error {
			list.ID = 1
			return nil
		},
	}
	service := NewTaskListService(store, &mockTaskItemStore{})

	list, err := service.CreateList(1, "Compras da semana")

	require.NoError(t, err)
	assert.Equal(t, uint(1), list.ID)
}

func TestTaskListService_CreateList_EmptyTitleReturnsError(t *testing.T) {
	service := NewTaskListService(&mockTaskListStore{}, &mockTaskItemStore{})

	_, err := service.CreateList(1, "")

	assert.EqualError(t, err, "título é obrigatório")
}

func TestTaskListService_DeleteList_RemovesItemsBeforeList(t *testing.T) {
	var deletedItemsForList uint
	var deletedList bool

	store := &mockTaskListStore{
		findByIDFunc: func(id, userID uint) (*models.TaskList, error) {
			return &models.TaskList{Model: gorm.Model{ID: id}, UserID: userID}, nil
		},
		deleteFunc: func(list *models.TaskList) error {
			deletedList = true
			return nil
		},
	}
	itemStore := &mockTaskItemStore{
		deleteAllByListFunc: func(taskListID uint) error {
			deletedItemsForList = taskListID
			return nil
		},
	}
	service := NewTaskListService(store, itemStore)

	require.NoError(t, service.DeleteList(5, 1))
	assert.Equal(t, uint(5), deletedItemsForList)
	assert.True(t, deletedList)
}

func TestTaskListService_DeleteList_NotFoundReturnsError(t *testing.T) {
	store := &mockTaskListStore{
		findByIDFunc: func(id, userID uint) (*models.TaskList, error) {
			return nil, errors.New("record not found")
		},
	}
	service := NewTaskListService(store, &mockTaskItemStore{})

	err := service.DeleteList(5, 1)

	assert.EqualError(t, err, "lista não encontrada")
}

func TestTaskListService_UpdateItem_Success(t *testing.T) {
	store := &mockTaskListStore{
		findByIDFunc: func(id, userID uint) (*models.TaskList, error) {
			return &models.TaskList{Model: gorm.Model{ID: id}, UserID: userID}, nil
		},
	}
	itemStore := &mockTaskItemStore{
		findByIDAndListFunc: func(id, taskListID uint) (*models.TaskItem, error) {
			return &models.TaskItem{Model: gorm.Model{ID: id}, TaskListID: taskListID}, nil
		},
		updateFunc: func(item *models.TaskItem) error { return nil },
	}
	service := NewTaskListService(store, itemStore)

	item, err := service.UpdateItem(1, 10, 1, "Leite desnatado", true)

	require.NoError(t, err)
	assert.Equal(t, "Leite desnatado", item.Text)
	assert.True(t, item.Completed)
}

func TestTaskListService_UpdateItem_ListNotOwnedByUserReturnsError(t *testing.T) {
	store := &mockTaskListStore{
		findByIDFunc: func(id, userID uint) (*models.TaskList, error) {
			return nil, errors.New("record not found")
		},
	}
	service := NewTaskListService(store, &mockTaskItemStore{})

	_, err := service.UpdateItem(1, 10, 999, "Tentando editar item de outro usuário", false)

	assert.EqualError(t, err, "lista não encontrada")
}
