package services

import (
	"errors"
	"math"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
	"github.com/RenanAlvesBCC/todolist-api/internal/repository"
)

// TaskListStore descreve o que o TaskListService precisa do repository de listas.
type TaskListStore interface {
	Create(list *models.TaskList) error
	FindAll(userID uint, workspaceID *uint, filter repository.TaskListFilter) ([]models.TaskList, int64, error)
	FindByIDAndUser(id, userID uint) (*models.TaskList, error)
	FindByID(id uint) (*models.TaskList, error)
	Update(list *models.TaskList) error
	Delete(list *models.TaskList) error
	NextPosition(userID uint) (int, error)
	UpdatePositions(userID uint, orderedIDs []uint) error
}

type TaskItemStore interface {
	Create(item *models.TaskItem) error
	FindByIDAndList(id, taskListID uint) (*models.TaskItem, error)
	Update(item *models.TaskItem) error
	Delete(item *models.TaskItem) error
	DeleteAllByList(taskListID uint) error
	NextPosition(taskListID uint) (int, error)
	UpdatePositions(taskListID uint, orderedIDs []uint) error
}

// WorkspaceContextStore é o subconjunto da interface de workspace que o TaskListService precisa.
type WorkspaceContextStore interface {
	FindByMemberUserID(userID uint) (*models.Workspace, error)
	GetMemberRole(workspaceID, userID uint) (models.WorkspaceRole, error)
	IsMember(workspaceID, userID uint) (bool, error)
}

type TaskListService struct {
	listRepo TaskListStore
	itemRepo TaskItemStore
	wsStore  WorkspaceContextStore
}

func NewTaskListService(listRepo TaskListStore, itemRepo TaskItemStore, wsStore WorkspaceContextStore) *TaskListService {
	return &TaskListService{listRepo: listRepo, itemRepo: itemRepo, wsStore: wsStore}
}

func (s *TaskListService) CreateList(userID uint, title string) (*models.TaskList, error) {
	if title == "" {
		return nil, errors.New("título é obrigatório")
	}

	position, err := s.listRepo.NextPosition(userID)
	if err != nil {
		return nil, err
	}

	list := &models.TaskList{
		Title:    title,
		UserID:   userID,
		Position: position,
		Status:   models.StatusEmAndamento,
		Items:    []models.TaskItem{},
	}
	if err := s.listRepo.Create(list); err != nil {
		return nil, err
	}
	return list, nil
}

type PaginatedTaskLists struct {
	Lists      []models.TaskList `json:"lists"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	Total      int64             `json:"total"`
	TotalPages int               `json:"total_pages"`
}

func (s *TaskListService) ListAll(userID uint, search string, page, limit int) (*PaginatedTaskLists, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var wsID *uint
	if s.wsStore != nil {
		if ws, err := s.wsStore.FindByMemberUserID(userID); err == nil {
			wsID = &ws.ID
		}
	}

	filter := repository.TaskListFilter{Search: search, Page: page, Limit: limit}
	lists, total, err := s.listRepo.FindAll(userID, wsID, filter)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return &PaginatedTaskLists{
		Lists:      lists,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// resolveList encontra uma lista verificando acesso do usuário (dono direto ou membro do workspace).
func (s *TaskListService) resolveList(listID, userID uint) (*models.TaskList, error) {
	list, err := s.listRepo.FindByID(listID)
	if err != nil {
		return nil, errors.New("lista não encontrada")
	}

	if list.UserID == userID {
		return list, nil
	}

	if list.WorkspaceID != nil && s.wsStore != nil {
		if ok, _ := s.wsStore.IsMember(*list.WorkspaceID, userID); ok {
			return list, nil
		}
	}

	return nil, errors.New("lista não encontrada")
}

func (s *TaskListService) GetList(listID, userID uint) (*models.TaskList, error) {
	return s.resolveList(listID, userID)
}

func (s *TaskListService) UpdateList(listID, userID uint, title string) (*models.TaskList, error) {
	if title == "" {
		return nil, errors.New("título é obrigatório")
	}

	list, err := s.listRepo.FindByIDAndUser(listID, userID)
	if err != nil {
		return nil, errors.New("lista não encontrada")
	}

	list.Title = title
	if err := s.listRepo.Update(list); err != nil {
		return nil, err
	}
	return list, nil
}

func (s *TaskListService) DeleteList(listID, userID uint) error {
	list, err := s.listRepo.FindByIDAndUser(listID, userID)
	if err != nil {
		return errors.New("lista não encontrada")
	}

	if err := s.itemRepo.DeleteAllByList(list.ID); err != nil {
		return err
	}
	return s.listRepo.Delete(list)
}

func (s *TaskListService) AddItem(listID, userID uint, text string) (*models.TaskItem, error) {
	if text == "" {
		return nil, errors.New("texto do item é obrigatório")
	}

	list, err := s.resolveList(listID, userID)
	if err != nil {
		return nil, err
	}

	position, err := s.itemRepo.NextPosition(list.ID)
	if err != nil {
		return nil, err
	}

	item := &models.TaskItem{Text: text, TaskListID: list.ID, Position: position}
	if err := s.itemRepo.Create(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *TaskListService) UpdateItem(listID, itemID, userID uint, text string, completed bool) (*models.TaskItem, error) {
	if text == "" {
		return nil, errors.New("texto do item é obrigatório")
	}

	list, err := s.resolveList(listID, userID)
	if err != nil {
		return nil, err
	}

	item, err := s.itemRepo.FindByIDAndList(itemID, list.ID)
	if err != nil {
		return nil, errors.New("item não encontrado")
	}

	item.Text = text
	item.Completed = completed
	if err := s.itemRepo.Update(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *TaskListService) DeleteItem(listID, itemID, userID uint) error {
	list, err := s.listRepo.FindByIDAndUser(listID, userID)
	if err != nil {
		return errors.New("lista não encontrada")
	}

	item, err := s.itemRepo.FindByIDAndList(itemID, list.ID)
	if err != nil {
		return errors.New("item não encontrado")
	}

	return s.itemRepo.Delete(item)
}

func (s *TaskListService) ReorderLists(userID uint, orderedIDs []uint) error {
	if len(orderedIDs) == 0 {
		return errors.New("lista de ids vazia")
	}
	return s.listRepo.UpdatePositions(userID, orderedIDs)
}

func (s *TaskListService) ReorderItems(listID, userID uint, orderedIDs []uint) error {
	list, err := s.listRepo.FindByIDAndUser(listID, userID)
	if err != nil {
		return errors.New("lista não encontrada")
	}
	if len(orderedIDs) == 0 {
		return errors.New("lista de ids vazia")
	}
	return s.itemRepo.UpdatePositions(list.ID, orderedIDs)
}

// validTransitions define as transições permitidas por papel.
var editorTransitions = map[models.TaskListStatus]map[models.TaskListStatus]bool{
	models.StatusEmAndamento:         {models.StatusAguardandoOrcamento: true, models.StatusAguardandoPeca: true},
	models.StatusAguardandoOrcamento: {models.StatusEmAndamento: true, models.StatusAguardandoPeca: true},
	models.StatusAguardandoPeca:      {models.StatusEmAndamento: true, models.StatusAguardandoOrcamento: true},
}

func (s *TaskListService) ChangeStatus(listID, userID uint, newStatus models.TaskListStatus) error {
	list, err := s.resolveList(listID, userID)
	if err != nil {
		return err
	}

	role, err := s.getUserRole(list, userID)
	if err != nil {
		return err
	}

	if !isValidTransition(role, list.Status, newStatus) {
		return errors.New("transição de status não permitida para seu papel")
	}

	list.Status = newStatus
	return s.listRepo.Update(list)
}

func (s *TaskListService) AssignMember(listID, requesterID, targetUserID uint) error {
	list, err := s.resolveList(listID, requesterID)
	if err != nil {
		return err
	}

	role, err := s.getUserRole(list, requesterID)
	if err != nil {
		return err
	}

	if role != models.RoleOwner && role != models.RoleManager {
		return errors.New("apenas owner e manager podem atribuir membros")
	}

	if list.WorkspaceID == nil {
		return errors.New("atribuição só é possível em listas de workspace")
	}

	if s.wsStore != nil {
		targetRole, err := s.wsStore.GetMemberRole(*list.WorkspaceID, targetUserID)
		if err != nil {
			return errors.New("usuário não é membro do workspace")
		}
		if targetRole != models.RoleEditor {
			return errors.New("só é possível atribuir mecânicos (editor)")
		}
	}

	list.AssignedTo = &targetUserID
	return s.listRepo.Update(list)
}

// getUserRole retorna o papel do usuário em relação à lista.
// Para listas pessoais, o dono recebe papel owner implícito.
func (s *TaskListService) getUserRole(list *models.TaskList, userID uint) (models.WorkspaceRole, error) {
	if list.WorkspaceID != nil && s.wsStore != nil {
		return s.wsStore.GetMemberRole(*list.WorkspaceID, userID)
	}
	if list.UserID == userID {
		return models.RoleOwner, nil
	}
	return "", errors.New("acesso negado")
}

func isValidTransition(role models.WorkspaceRole, from, to models.TaskListStatus) bool {
	if role == models.RoleOwner || role == models.RoleManager {
		return true
	}
	allowed, ok := editorTransitions[from]
	return ok && allowed[to]
}
