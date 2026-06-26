package repository

import (
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

type PendingFlagRepository struct {
	db *gorm.DB
}

func NewPendingFlagRepository(db *gorm.DB) *PendingFlagRepository {
	return &PendingFlagRepository{db: db}
}

func (r *PendingFlagRepository) Create(f *models.PendingFlag) error {
	return r.db.Create(f).Error
}

func (r *PendingFlagRepository) FindByID(id uint) (*models.PendingFlag, error) {
	var f models.PendingFlag
	if err := r.db.First(&f, id).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *PendingFlagRepository) ListByTaskList(taskListID uint) ([]models.PendingFlag, error) {
	var flags []models.PendingFlag
	if err := r.db.Where("task_list_id = ?", taskListID).Order("created_at asc").Find(&flags).Error; err != nil {
		return nil, err
	}
	return flags, nil
}

func (r *PendingFlagRepository) Update(f *models.PendingFlag) error {
	return r.db.Save(f).Error
}
