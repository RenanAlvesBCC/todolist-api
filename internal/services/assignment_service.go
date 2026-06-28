package services

import (
	"errors"
	"time"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

type AssignmentStore interface {
	Assign(a *models.ListAssignment) error
	Unassign(taskListID, userID uint) error
	ListByTaskList(taskListID uint) ([]models.ListAssignment, error)
	IsAssigned(taskListID, userID uint) (bool, error)
	UnassignAll(taskListID uint) error
}

type AssignmentService struct {
	repo      AssignmentStore
	wsStore   WorkspaceStore
	listStore TaskListStore
}

func NewAssignmentService(repo AssignmentStore, wsStore WorkspaceStore, listStore TaskListStore) *AssignmentService {
	return &AssignmentService{repo: repo, wsStore: wsStore, listStore: listStore}
}

// Assign atribui um mecânico a um veículo.
// Só owner e manager podem atribuir.
// O usuário alvo deve ser membro do workspace.
func (s *AssignmentService) Assign(requesterID, taskListID, targetUserID uint) error {
	ws, err := s.wsStore.FindByMemberUserID(requesterID)
	if err != nil {
		return errors.New("workspace não encontrado")
	}

	role, err := s.wsStore.GetMemberRole(ws.ID, requesterID)
	if err != nil || (role != models.RoleOwner && role != models.RoleManager) {
		return errors.New("apenas gerentes podem atribuir mecânicos")
	}

	isMember, _ := s.wsStore.IsMember(ws.ID, targetUserID)
	if !isMember {
		return errors.New("usuário não é membro do workspace")
	}

	already, _ := s.repo.IsAssigned(taskListID, targetUserID)
	if already {
		return errors.New("mecânico já está atribuído a este veículo")
	}

	return s.repo.Assign(&models.ListAssignment{
		TaskListID: taskListID,
		UserID:     targetUserID,
		AssignedBy: requesterID,
		AssignedAt: time.Now(),
	})
}

func (s *AssignmentService) Unassign(requesterID, taskListID, targetUserID uint) error {
	ws, err := s.wsStore.FindByMemberUserID(requesterID)
	if err != nil {
		return errors.New("workspace não encontrado")
	}

	role, err := s.wsStore.GetMemberRole(ws.ID, requesterID)
	if err != nil || (role != models.RoleOwner && role != models.RoleManager) {
		return errors.New("apenas gerentes podem remover atribuições")
	}

	return s.repo.Unassign(taskListID, targetUserID)
}

func (s *AssignmentService) List(taskListID uint) ([]models.ListAssignment, error) {
	return s.repo.ListByTaskList(taskListID)
}
