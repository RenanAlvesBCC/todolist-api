package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

type SecurityRepository struct {
	db *gorm.DB
}

func NewSecurityRepository(db *gorm.DB) *SecurityRepository {
	return &SecurityRepository{db: db}
}

// BlacklistToken adiciona um token à lista negra até ele expirar naturalmente.
func (r *SecurityRepository) BlacklistToken(token string, expiresAt time.Time) error {
	entry := &models.TokenBlacklist{Token: token, ExpiresAt: expiresAt}
	return r.db.Create(entry).Error
}

// IsTokenBlacklisted verifica se um token foi revogado.
func (r *SecurityRepository) IsTokenBlacklisted(token string) (bool, error) {
	var count int64
	err := r.db.Model(&models.TokenBlacklist{}).
		Where("token = ? AND expires_at > ?", token, time.Now()).
		Count(&count).Error
	return count > 0, err
}

// CleanExpiredTokens remove tokens já expirados da blacklist
// (chamado periodicamente pra não deixar a tabela crescer infinitamente).
func (r *SecurityRepository) CleanExpiredTokens() error {
	return r.db.Where("expires_at <= ?", time.Now()).Delete(&models.TokenBlacklist{}).Error
}

// LogAction registra uma ação de auditoria.
func (r *SecurityRepository) LogAction(userID *uint, action, ip, userAgent, details string, success bool) {
	entry := &models.AuditLog{
		UserID:    userID,
		Action:    action,
		IPAddress: ip,
		UserAgent: userAgent,
		Details:   details,
		Success:   success,
	}
	// Usamos go routine pra não bloquear a requisição principal
	// se o log falhar por algum motivo
	go func() {
		r.db.Create(entry)
	}()
}
