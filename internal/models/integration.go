package models

import "time"

const (
	SyncStatusRunning = "running"
	SyncStatusSuccess = "success"
	SyncStatusPartial = "partial"
	SyncStatusError   = "error"
)

const (
	SyncTriggerSchedule = "schedule"
	SyncTriggerManual   = "manual"
	SyncTriggerUpload   = "upload"
)

// SyncRun é uma execução da importação OnMotor (pode chegar em vários lotes).
// Warnings nunca contêm dados pessoais: só ids e contagens.
type SyncRun struct {
	ID               uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	WorkspaceID      uint       `gorm:"not null;index" json:"workspace_id"`
	Source           string     `gorm:"not null;default:'onmotor'" json:"source"`
	StartedAt        time.Time  `gorm:"not null" json:"started_at"`
	FinishedAt       *time.Time `json:"finished_at"`
	Status           string     `gorm:"not null;default:'running'" json:"status"`
	Trigger          string     `gorm:"not null;default:'manual'" json:"trigger"`
	CustomersCreated int        `gorm:"not null;default:0" json:"customers_created"`
	CustomersUpdated int        `gorm:"not null;default:0" json:"customers_updated"`
	VehiclesCreated  int        `gorm:"not null;default:0" json:"vehicles_created"`
	VehiclesUpdated  int        `gorm:"not null;default:0" json:"vehicles_updated"`
	ServicesCreated  int        `gorm:"not null;default:0" json:"services_created"`
	ServicesUpdated  int        `gorm:"not null;default:0" json:"services_updated"`
	Skipped          int        `gorm:"not null;default:0" json:"skipped"`
	Warnings         StringList `gorm:"not null;default:'[]'" json:"warnings"`
	Error            string     `gorm:"not null;default:''" json:"error"`
}

// IntegrationToken autentica o robô de sincronização. Só o hash SHA-256 fica no banco;
// o token em texto aparece uma única vez, na criação.
type IntegrationToken struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	WorkspaceID uint       `gorm:"not null;index" json:"workspace_id"`
	Name        string     `gorm:"not null" json:"name"`
	TokenHash   string     `gorm:"not null;uniqueIndex" json:"-"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedBy   uint       `gorm:"not null" json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
}
