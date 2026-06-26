package models

import "time"

type QuoteItem struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskListID  uint      `gorm:"not null;index" json:"task_list_id"`
	SubmittedBy uint      `gorm:"not null" json:"submitted_by"`
	Text        string    `gorm:"not null" json:"text"`
	CreatedAt   time.Time `json:"created_at"`
}
