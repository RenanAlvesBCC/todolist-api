package models

import "time"

// TokenBlacklist guarda tokens JWT revogados antes da expiração natural.
// Quando um usuário faz logout, o token entra aqui e passa a ser rejeitado
// pelo middleware mesmo que ainda não tenha expirado.
type TokenBlacklist struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Token     string    `gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"not null;index"`
	CreatedAt time.Time
}
