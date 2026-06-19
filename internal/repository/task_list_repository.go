package repository

import (
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

type TaskListRepository struct {
	db *gorm.DB
}

func NewTaskListRepository(db *gorm.DB) *TaskListRepository {
	return &TaskListRepository{db: db}
}

type TaskListFilter struct {
	Search string
	Page   int
	Limit  int
}

func (r *TaskListRepository) applyFilters(userID uint, filter TaskListFilter) *gorm.DB {
	query := r.db.Model(&models.TaskList{}).Where("user_id = ?", userID)
	if filter.Search != "" {
		query = query.Where("title LIKE ?", "%"+filter.Search+"%")
	}
	return query
}

func (r *TaskListRepository) Create(list *models.TaskList) error {
	return r.db.Create(list).Error
}

func (r *TaskListRepository) FindAllByUser(userID uint, filter TaskListFilter) ([]models.TaskList, int64, error) {
	var lists []models.TaskList
	var total int64

	if err := r.applyFilters(userID, filter).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.Limit
	if err := r.applyFilters(userID, filter).
		Preload("Items").
		Order("created_at desc").
		Offset(offset).Limit(filter.Limit).
		Find(&lists).Error; err != nil {
		return nil, 0, err
	}

	return lists, total, nil
}

func (r *TaskListRepository) FindByIDAndUser(id, userID uint) (*models.TaskList, error) {
	var list models.TaskList
	if err := r.db.Preload("Items").Where("id = ? AND user_id = ?", id, userID).First(&list).Error; err != nil {
		return nil, err
	}
	return &list, nil
}

func (r *TaskListRepository) Update(list *models.TaskList) error {
	return r.db.Save(list).Error
}

func (r *TaskListRepository) Delete(list *models.TaskList) error {
	return r.db.Delete(list).Error
}
