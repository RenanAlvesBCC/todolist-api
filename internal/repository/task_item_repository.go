package repository

import (
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

type TaskItemRepository struct {
	db *gorm.DB
}

func NewTaskItemRepository(db *gorm.DB) *TaskItemRepository {
	return &TaskItemRepository{db: db}
}

func (r *TaskItemRepository) Create(item *models.TaskItem) error {
	return r.db.Create(item).Error
}

func (r *TaskItemRepository) FindByIDAndList(id, taskListID uint) (*models.TaskItem, error) {
	var item models.TaskItem
	if err := r.db.Where("id = ? AND task_list_id = ?", id, taskListID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *TaskItemRepository) Update(item *models.TaskItem) error {
	return r.db.Save(item).Error
}

func (r *TaskItemRepository) Delete(item *models.TaskItem) error {
	return r.db.Delete(item).Error
}

func (r *TaskItemRepository) DeleteAllByList(taskListID uint) error {
	return r.db.Where("task_list_id = ?", taskListID).Delete(&models.TaskItem{}).Error
}
