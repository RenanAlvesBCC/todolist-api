package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

// Nome do autor: "first_name last_name" sem espaços nas pontas; contas antigas sem nome caem no username.
const noteAuthorNameSelect = "(SELECT COALESCE(NULLIF(TRIM(users.first_name || ' ' || users.last_name), ''), users.username)" +
	" FROM users WHERE users.id = vehicle_notes.author_id) AS author_name"

// notesWithAuthorNewestFirst é usado tanto no endpoint de notas quanto no Preload de TaskList.Notes.
func notesWithAuthorNewestFirst(db *gorm.DB) *gorm.DB {
	return db.
		Select("vehicle_notes.*, " + noteAuthorNameSelect).
		Order("vehicle_notes.created_at DESC").
		Order("vehicle_notes.id DESC")
}

type VehicleNoteRepository struct {
	db *gorm.DB
}

func NewVehicleNoteRepository(db *gorm.DB) *VehicleNoteRepository {
	return &VehicleNoteRepository{db: db}
}

// Create grava a nota e a recarrega para devolver author_name preenchido.
func (r *VehicleNoteRepository) Create(n *models.VehicleNote) error {
	if err := r.db.Create(n).Error; err != nil {
		return err
	}
	return r.db.Scopes(notesWithAuthorNewestFirst).First(n, n.ID).Error
}

func (r *VehicleNoteRepository) FindByID(id uint) (*models.VehicleNote, error) {
	var n models.VehicleNote
	if err := r.db.Scopes(notesWithAuthorNewestFirst).First(&n, id).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *VehicleNoteRepository) ListByTaskList(taskListID uint) ([]models.VehicleNote, error) {
	notes := []models.VehicleNote{}
	if err := r.db.Scopes(notesWithAuthorNewestFirst).
		Where("vehicle_notes.task_list_id = ?", taskListID).
		Find(&notes).Error; err != nil {
		return nil, err
	}
	return notes, nil
}

// SoftDelete marca a nota como excluída (deleted_at + deleted_by) sem apagar a linha.
func (r *VehicleNoteRepository) SoftDelete(n *models.VehicleNote, deletedBy uint) error {
	result := r.db.Model(&models.VehicleNote{}).
		Where("id = ?", n.ID).
		UpdateColumns(map[string]interface{}{
			"deleted_at": time.Now().UTC(),
			"deleted_by": deletedBy,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
