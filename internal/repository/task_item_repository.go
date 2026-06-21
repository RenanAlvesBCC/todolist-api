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

func (r *TaskItemRepository) NextPosition(taskListID uint) (int, error) {
	var count int64
	if err := r.db.Model(&models.TaskItem{}).Where("task_list_id = ?", taskListID).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r *TaskItemRepository) UpdatePositions(taskListID uint, orderedIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for index, id := range orderedIDs {
			result := tx.Model(&models.TaskItem{}).
				Where("id = ? AND task_list_id = ?", id, taskListID).
				Update("position", index)
			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
}
