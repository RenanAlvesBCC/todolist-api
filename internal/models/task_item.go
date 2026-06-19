package models

import "gorm.io/gorm"

// TaskItem representa um item marcável dentro de uma TaskList
// (ex: "Leite" dentro do bloco "Compras da semana").
type TaskItem struct {
	gorm.Model
	Text       string `gorm:"not null" json:"text"`
	Completed  bool   `gorm:"default:false" json:"completed"`
	TaskListID uint   `gorm:"not null" json:"task_list_id"`
}
