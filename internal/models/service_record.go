package models

import "time"

// ServiceRecord é uma OS do histórico externo (OnMotor). Nunca é escrita de volta lá.
type ServiceRecord struct {
	ID                uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	WorkspaceID       uint       `gorm:"not null;uniqueIndex:idx_service_records_source" json:"workspace_id"`
	Source            string     `gorm:"not null;default:'onmotor';uniqueIndex:idx_service_records_source" json:"source"`
	SourceID          string     `gorm:"not null;uniqueIndex:idx_service_records_source" json:"source_id"`
	CustomerID        *uint      `gorm:"index" json:"customer_id"`
	CustomerVehicleID *uint      `gorm:"index" json:"customer_vehicle_id"`
	Plate             string     `gorm:"not null;default:'';index" json:"plate"`
	Mileage           *int       `json:"mileage"`
	Type              string     `gorm:"not null;default:''" json:"type"`
	Status            string     `gorm:"not null;default:''" json:"status"`
	OpenedAt          *time.Time `json:"opened_at"`
	DiagnosedAt       *time.Time `json:"diagnosed_at"`
	QuotedAt          *time.Time `json:"quoted_at"`
	StartedAt         *time.Time `json:"started_at"`
	TestedAt          *time.Time `json:"tested_at"`
	FinishedAt        *time.Time `json:"finished_at"`
	NotifiedAt        *time.Time `json:"notified_at"`
	DeliveredAt       *time.Time `json:"delivered_at"`
	Notes             string     `gorm:"not null;default:''" json:"notes"`
	// Array JSON de {kind, description, quantity, unit_price, total}; formato flexível.
	Items     JSONText   `gorm:"not null;default:'[]'" json:"items"`
	Total     *float64   `gorm:"type:numeric(14,2)" json:"total"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	SyncedAt  *time.Time `json:"synced_at"`
}
