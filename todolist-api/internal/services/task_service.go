package services

import (
	"errors"

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

func (s *TaskService) List(userID uint) ([]models.Task, error) {
	return s.taskRepo.FindAllByUser(userID)
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
