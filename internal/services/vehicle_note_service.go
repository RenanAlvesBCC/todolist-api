package services

import (
	"errors"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

const MaxVehicleNoteLength = 1000

var (
	ErrNoteTextRequired = errors.New("texto é obrigatório")
	ErrNoteTextTooLong  = errors.New("texto deve ter no máximo 1000 caracteres")
	ErrNoteNotFound     = errors.New("informação não encontrada")
	ErrNoteForbidden    = errors.New("apenas gerentes ou o autor podem excluir esta informação")
)

type VehicleNoteStore interface {
	Create(n *models.VehicleNote) error
	FindByID(id uint) (*models.VehicleNote, error)
	ListByTaskList(taskListID uint) ([]models.VehicleNote, error)
	SoftDelete(n *models.VehicleNote, deletedBy uint) error
}

type VehicleNoteService struct {
	store    VehicleNoteStore
	listRepo TaskListStore
	wsStore  WorkspaceContextStore
}

func NewVehicleNoteService(store VehicleNoteStore, listRepo TaskListStore, wsStore WorkspaceContextStore) *VehicleNoteService {
	return &VehicleNoteService{store: store, listRepo: listRepo, wsStore: wsStore}
}

func (s *VehicleNoteService) AddNote(listID, userID uint, text string) (*models.VehicleNote, error) {
	if _, err := s.resolveList(listID, userID); err != nil {
		return nil, err
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrNoteTextRequired
	}
	if utf8.RuneCountInString(text) > MaxVehicleNoteLength {
		return nil, ErrNoteTextTooLong
	}

	n := &models.VehicleNote{TaskListID: listID, AuthorID: userID, Text: text}
	if err := s.store.Create(n); err != nil {
		return nil, err
	}
	return n, nil
}

func (s *VehicleNoteService) ListNotes(listID, userID uint) ([]models.VehicleNote, error) {
	if _, err := s.resolveList(listID, userID); err != nil {
		return nil, err
	}
	return s.store.ListByTaskList(listID)
}

// DeleteNote faz a exclusão lógica. Dono/gerente excluem qualquer nota; os demais, só as próprias.
func (s *VehicleNoteService) DeleteNote(listID, noteID, userID uint) error {
	list, err := s.resolveList(listID, userID)
	if err != nil {
		return err
	}

	n, err := s.store.FindByID(noteID)
	if err != nil || n.TaskListID != listID {
		return ErrNoteNotFound
	}

	if n.AuthorID != userID {
		role, err := vehicleRole(list, userID, s.wsStore)
		if err != nil || (role != models.RoleOwner && role != models.RoleManager) {
			return ErrNoteForbidden
		}
	}

	if err := s.store.SoftDelete(n, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNoteNotFound
		}
		return err
	}
	return nil
}

func (s *VehicleNoteService) resolveList(listID, userID uint) (*models.TaskList, error) {
	list, err := s.listRepo.FindByID(listID)
	if err != nil {
		return nil, ErrNotFound
	}
	if err := checkVehicleAccess(list, userID, s.wsStore); err != nil {
		return nil, err
	}
	return list, nil
}
