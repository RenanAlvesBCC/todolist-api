package models

import "time"

type ListAssignment struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskListID uint      `gorm:"not null;uniqueIndex:idx_list_user" json:"task_list_id"`
	UserID     uint      `gorm:"not null;uniqueIndex:idx_list_user" json:"user_id"`
	AssignedBy uint      `gorm:"not null" json:"assigned_by"`
	AssignedAt time.Time `json:"assigned_at"`
}

type AssignmentUser struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
}
