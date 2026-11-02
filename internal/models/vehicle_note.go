package models

import (
	"time"

	"gorm.io/gorm"
)

// VehicleNote é uma "informação importante" do veículo. A exclusão é lógica:
// a linha fica no banco com deleted_at/deleted_by e o GORM a esconde das consultas.
type VehicleNote struct {
	ID         uint `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskListID uint `gorm:"not null;index" json:"task_list_id"`
	AuthorID   uint `gorm:"not null;index" json:"author_id"`
	// Calculado na consulta a partir de users; não é coluna de vehicle_notes.
	AuthorName string         `gorm:"-:migration;->" json:"author_name"`
	Text       string         `gorm:"not null" json:"text"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	DeletedBy  *uint          `json:"-"`
}
