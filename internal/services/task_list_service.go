package services

import (
	"errors"
	"math"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
	"github.com/RenanAlvesBCC/todolist-api/internal/repository"
)

// TaskListStore descreve o que o TaskListService precisa do repository de listas —
// pode ser o GORM de verdade ou um dublê de teste, contanto que tenha esses métodos.
type TaskListStore interface {
	Create(list *models.TaskList) error
	FindAllByUser(userID uint, filter repository.TaskListFilter) ([]models.TaskList, int64, error)
	FindByIDAndUser(id, userID uint) (*models.TaskList, error)
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

type TaskListService struct {
	listRepo TaskListStore
	itemRepo TaskItemStore
}

func NewTaskListService(listRepo TaskListStore, itemRepo TaskItemStore) *TaskListService {
	return &TaskListService{listRepo: listRepo, itemRepo: itemRepo}
}

func (s *TaskListService) CreateList(userID uint, title string) (*models.TaskList, error) {
	if title == "" {
		return nil, errors.New("título é obrigatório")
	}

	position, err := s.listRepo.NextPosition(userID)
	if err != nil {
		return nil, err
	}

	list := &models.TaskList{Title: title, UserID: userID, Position: position, Items: []models.TaskItem{}}
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

	filter := repository.TaskListFilter{Search: search, Page: page, Limit: limit}
	lists, total, err := s.listRepo.FindAllByUser(userID, filter)
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

func (s *TaskListService) GetList(listID, userID uint) (*models.TaskList, error) {
	list, err := s.listRepo.FindByIDAndUser(listID, userID)
	if err != nil {
		return nil, errors.New("lista não encontrada")
	}
	return list, nil
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

	list, err := s.listRepo.FindByIDAndUser(listID, userID)
	if err != nil {
		return nil, errors.New("lista não encontrada")
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

	list, err := s.listRepo.FindByIDAndUser(listID, userID)
	if err != nil {
		return nil, errors.New("lista não encontrada")
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
