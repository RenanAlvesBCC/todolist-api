package services

import (
	"errors"
	"math"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
	"github.com/RenanAlvesBCC/todolist-api/internal/repository"
)

type TaskService struct {
	taskRepo *repository.TaskRepository
}

func NewTaskService(taskRepo *repository.TaskRepository) *TaskService {
	return &TaskService{taskRepo: taskRepo}
}

func (s *TaskService) Create(userID uint, title, description string) (*models.Task, error) {
	if title == "" {
		return nil, errors.New("título é obrigatório")
	}

	task := &models.Task{
		Title:       title,
		Description: description,
		UserID:      userID,
	}

	if err := s.taskRepo.Create(task); err != nil {
		return nil, err
	}
	return task, nil
}

// PaginatedTasks é o formato devolvido pela listagem: além das tarefas,
// traz os metadados que o cliente (incluindo o futuro app SwiftUI)
// precisa pra montar a navegação entre páginas.
type PaginatedTasks struct {
	Tasks      []models.Task `json:"tasks"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	Total      int64         `json:"total"`
	TotalPages int           `json:"total_pages"`
}

func (s *TaskService) List(userID uint, completed *bool, search string, page, limit int) (*PaginatedTasks, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	filter := repository.TaskFilter{
		Completed: completed,
		Search:    search,
		Page:      page,
		Limit:     limit,
	}

	tasks, total, err := s.taskRepo.FindAllByUser(userID, filter)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return &PaginatedTasks{
		Tasks:      tasks,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *TaskService) Get(taskID, userID uint) (*models.Task, error) {
	task, err := s.taskRepo.FindByIDAndUser(taskID, userID)
	if err != nil {
		return nil, errors.New("tarefa não encontrada")
	}
	return task, nil
}

func (s *TaskService) Update(taskID, userID uint, title, description string, completed bool) (*models.Task, error) {
	task, err := s.taskRepo.FindByIDAndUser(taskID, userID)
	if err != nil {
		return nil, errors.New("tarefa não encontrada")
	}

	task.Title = title
	task.Description = description
	task.Completed = completed

	if err := s.taskRepo.Update(task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) Delete(taskID, userID uint) error {
	task, err := s.taskRepo.FindByIDAndUser(taskID, userID)
	if err != nil {
		return errors.New("tarefa não encontrada")
	}
	return s.taskRepo.Delete(task)
}
