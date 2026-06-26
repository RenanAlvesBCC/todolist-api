package services

import (
	"errors"
	"time"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

type PendingFlagStore interface {
	Create(f *models.PendingFlag) error
	FindByID(id uint) (*models.PendingFlag, error)
	ListByTaskList(taskListID uint) ([]models.PendingFlag, error)
	Update(f *models.PendingFlag) error
}

type PendingFlagService struct {
	store    PendingFlagStore
	listRepo TaskListStore
	wsStore  WorkspaceContextStore
}

func NewPendingFlagService(store PendingFlagStore, listRepo TaskListStore, wsStore WorkspaceContextStore) *PendingFlagService {
	return &PendingFlagService{store: store, listRepo: listRepo, wsStore: wsStore}
}

func (s *PendingFlagService) AddFlag(listID, userID uint, flagType models.FlagType, note string) (*models.PendingFlag, error) {
	if err := s.checkListAccess(listID, userID); err != nil {
		return nil, err
	}

	if flagType == models.FlagOutro && len([]rune(note)) < 10 {
		return nil, errors.New("nota obrigatória com mínimo de 10 caracteres para tipo 'outro'")
	}

	f := &models.PendingFlag{
		TaskListID: listID,
		CreatedBy:  userID,
		FlagType:   flagType,
		Note:       note,
		CreatedAt:  time.Now(),
	}
	if err := s.store.Create(f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *PendingFlagService) ResolveFlag(listID, flagID, userID uint) error {
	if err := s.checkListAccess(listID, userID); err != nil {
		return err
	}

	f, err := s.store.FindByID(flagID)
	if err != nil || f.TaskListID != listID {
		return errors.New("pendência não encontrada")
	}

	if f.ResolvedAt != nil {
		return errors.New("pendência já resolvida")
	}

	now := time.Now()
	f.ResolvedAt = &now
	f.ResolvedBy = &userID
	return s.store.Update(f)
}

func (s *PendingFlagService) ListFlags(listID, userID uint) ([]models.PendingFlag, error) {
	if err := s.checkListAccess(listID, userID); err != nil {
		return nil, err
	}
	return s.store.ListByTaskList(listID)
}

func (s *PendingFlagService) checkListAccess(listID, userID uint) error {
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
