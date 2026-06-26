package models

import "gorm.io/gorm"

type TaskListStatus string

const (
	StatusEmAndamento         TaskListStatus = "em_andamento"
	StatusAguardandoOrcamento TaskListStatus = "aguardando_orcamento"
	StatusAguardandoPeca      TaskListStatus = "aguardando_peca"
	StatusAprovado            TaskListStatus = "aprovado"
	StatusConcluido           TaskListStatus = "concluido"
)

type TaskList struct {
	gorm.Model
	Title       string         `gorm:"not null" json:"title"`
	UserID      uint           `gorm:"not null" json:"user_id"`
	WorkspaceID *uint          `json:"workspace_id"`
	AssignedTo  *uint          `json:"assigned_to"`
	Status      TaskListStatus `gorm:"not null;default:'em_andamento'" json:"status"`
	Position    int            `gorm:"not null;default:0" json:"position"`
	Items       []TaskItem     `gorm:"foreignKey:TaskListID" json:"items"`
}
