package repository

import (
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

type ListAssignmentRepository struct {
	db *gorm.DB
}

func NewListAssignmentRepository(db *gorm.DB) *ListAssignmentRepository {
	return &ListAssignmentRepository{db: db}
}

func (r *ListAssignmentRepository) Assign(a *models.ListAssignment) error {
	return r.db.Create(a).Error
}

func (r *ListAssignmentRepository) Unassign(taskListID, userID uint) error {
	return r.db.Where("task_list_id = ? AND user_id = ?", taskListID, userID).
		Delete(&models.ListAssignment{}).Error
}

func (r *ListAssignmentRepository) ListByTaskList(taskListID uint) ([]models.ListAssignment, error) {
	var assignments []models.ListAssignment
	err := r.db.Where("task_list_id = ?", taskListID).Find(&assignments).Error
	return assignments, err
}

func (r *ListAssignmentRepository) IsAssigned(taskListID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.ListAssignment{}).
		Where("task_list_id = ? AND user_id = ?", taskListID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *ListAssignmentRepository) UnassignAll(taskListID uint) error {
	return r.db.Where("task_list_id = ?", taskListID).
		Delete(&models.ListAssignment{}).Error
}
