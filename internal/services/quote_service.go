package services

import (
	"errors"
	"time"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

type QuoteStore interface {
	Create(q *models.QuoteItem) error
	FindByID(id uint) (*models.QuoteItem, error)
	ListByTaskList(taskListID uint) ([]models.QuoteItem, error)
	Delete(q *models.QuoteItem) error
}

type QuoteService struct {
	store    QuoteStore
	listRepo TaskListStore
	wsStore  WorkspaceContextStore
}

func NewQuoteService(store QuoteStore, listRepo TaskListStore, wsStore WorkspaceContextStore) *QuoteService {
	return &QuoteService{store: store, listRepo: listRepo, wsStore: wsStore}
}

func (s *QuoteService) AddQuote(listID, userID uint, text string) (*models.QuoteItem, error) {
	if text == "" {
		return nil, errors.New("texto é obrigatório")
	}

	if err := s.checkListAccess(listID, userID); err != nil {
		return nil, err
	}

	q := &models.QuoteItem{
		TaskListID:  listID,
		SubmittedBy: userID,
		Text:        text,
		CreatedAt:   time.Now(),
	}
	if err := s.store.Create(q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *QuoteService) ListQuotes(listID, userID uint) ([]models.QuoteItem, error) {
	if err := s.checkListAccess(listID, userID); err != nil {
		return nil, err
	}
	return s.store.ListByTaskList(listID)
}

func (s *QuoteService) DeleteQuote(listID, quoteID, userID uint) error {
	if err := s.checkListAccess(listID, userID); err != nil {
		return err
	}

	q, err := s.store.FindByID(quoteID)
	if err != nil || q.TaskListID != listID {
		return errors.New("orçamento não encontrado")
	}
	return s.store.Delete(q)
}

func (s *QuoteService) checkListAccess(listID, userID uint) error {
	list, err := s.listRepo.FindByID(listID)
	if err != nil {
		return errors.New("lista não encontrada")
	}

	if list.UserID == userID {
		return nil
	}

	if list.WorkspaceID != nil && s.wsStore != nil {
		if ok, _ := s.wsStore.IsMember(*list.WorkspaceID, userID); ok {
			return nil
		}
	}

	return errors.New("lista não encontrada")
}
