package models

import "gorm.io/gorm"

// TaskList representa um bloco de tarefas — o "card" do seu rascunho —
// pertencente a um usuário e contendo vários itens marcáveis dentro.
type TaskList struct {
	gorm.Model
	Title  string     `gorm:"not null" json:"title"`
	UserID uint       `gorm:"not null" json:"user_id"`
	Items  []TaskItem `gorm:"foreignKey:TaskListID" json:"items"`
}
