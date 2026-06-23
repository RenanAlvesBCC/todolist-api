package models

import "time"

// AuditLog registra ações sensíveis: logins, logouts, tentativas falhas,
// criação/deleção de dados. Útil pra detectar padrões suspeitos.
type AuditLog struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	UserID    *uint  `gorm:"index"`
	Action    string `gorm:"not null"`
	IPAddress string `gorm:"not null"`
	UserAgent string
	Details   string
	Success   bool      `gorm:"not null"`
	CreatedAt time.Time `gorm:"index"`
}
