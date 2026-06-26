package repository

import (
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

type QuoteRepository struct {
	db *gorm.DB
}

func NewQuoteRepository(db *gorm.DB) *QuoteRepository {
	return &QuoteRepository{db: db}
}

func (r *QuoteRepository) Create(q *models.QuoteItem) error {
	return r.db.Create(q).Error
}

func (r *QuoteRepository) FindByID(id uint) (*models.QuoteItem, error) {
	var q models.QuoteItem
	if err := r.db.First(&q, id).Error; err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *QuoteRepository) ListByTaskList(taskListID uint) ([]models.QuoteItem, error) {
	var items []models.QuoteItem
	if err := r.db.Where("task_list_id = ?", taskListID).Order("created_at asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *QuoteRepository) Delete(q *models.QuoteItem) error {
	return r.db.Delete(q).Error
}
