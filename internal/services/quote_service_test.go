package services

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

type mockQuoteStore struct {
	created  []models.QuoteItem
	findByID func(id uint) (*models.QuoteItem, error)
	list     []models.QuoteItem
	deleted  *models.QuoteItem
}

func (m *mockQuoteStore) Create(q *models.QuoteItem) error {
	q.ID = uint(len(m.created) + 1)
	m.created = append(m.created, *q)
	return nil
}
func (m *mockQuoteStore) FindByID(id uint) (*models.QuoteItem, error) {
	if m.findByID != nil {
		return m.findByID(id)
	}
	return nil, errors.New("not found")
}
func (m *mockQuoteStore) ListByTaskList(taskListID uint) ([]models.QuoteItem, error) {
	return m.list, nil
}
func (m *mockQuoteStore) Delete(q *models.QuoteItem) error {
	m.deleted = q
	return nil
}

func quoteListStore(userID uint) *mockTaskListStore {
	return &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) {
			return &models.TaskList{Model: gorm.Model{ID: id}, UserID: userID}, nil
		},
	}
}

func TestQuoteService_AddQuote_Success(t *testing.T) {
	store := &mockQuoteStore{}
	svc := NewQuoteService(store, quoteListStore(1), nil)

	q, err := svc.AddQuote(1, 1, "Pastilha de freio R$ 120")

	require.NoError(t, err)
	assert.Equal(t, "Pastilha de freio R$ 120", q.Text)
	assert.Len(t, store.created, 1)
}

func TestQuoteService_AddQuote_EmptyTextReturnsError(t *testing.T) {
	svc := NewQuoteService(&mockQuoteStore{}, quoteListStore(1), nil)

	_, err := svc.AddQuote(1, 1, "")

	assert.EqualError(t, err, "texto é obrigatório")
}

func TestQuoteService_AddQuote_ListNotFoundReturnsError(t *testing.T) {
	listStore := &mockTaskListStore{
		findByIDFunc: func(id uint) (*models.TaskList, error) { return nil, errors.New("not found") },
	}
	svc := NewQuoteService(&mockQuoteStore{}, listStore, nil)

	_, err := svc.AddQuote(99, 1, "texto")

	assert.EqualError(t, err, "lista não encontrada")
}

func TestQuoteService_ListQuotes_Success(t *testing.T) {
	store := &mockQuoteStore{list: []models.QuoteItem{{ID: 1, Text: "Item A"}}}
	svc := NewQuoteService(store, quoteListStore(1), nil)

	items, err := svc.ListQuotes(1, 1)

	require.NoError(t, err)
	assert.Len(t, items, 1)
}

func TestQuoteService_DeleteQuote_Success(t *testing.T) {
	q := &models.QuoteItem{ID: 5, TaskListID: 1}
	store := &mockQuoteStore{
		findByID: func(id uint) (*models.QuoteItem, error) { return q, nil },
	}
	svc := NewQuoteService(store, quoteListStore(1), nil)

	err := svc.DeleteQuote(1, 5, 1)

	require.NoError(t, err)
	assert.Equal(t, q, store.deleted)
}

func TestQuoteService_DeleteQuote_WrongListReturnsError(t *testing.T) {
	// quote pertence à lista 99, mas requisição é para lista 1
	store := &mockQuoteStore{
		findByID: func(id uint) (*models.QuoteItem, error) {
			return &models.QuoteItem{ID: id, TaskListID: 99}, nil
		},
	}
	svc := NewQuoteService(store, quoteListStore(1), nil)

	err := svc.DeleteQuote(1, 5, 1)

	assert.EqualError(t, err, "orçamento não encontrado")
}
