package models

import "gorm.io/gorm"

// Task representa uma tarefa do To-Do List, sempre vinculada a um usuário.
type Task struct {
	gorm.Model
	Title       string `gorm:"not null" json:"title"`
	Description string `json:"description"`
	Completed   bool   `gorm:"default:false" json:"completed"`
	UserID      uint   `gorm:"not null" json:"user_id"`
}
