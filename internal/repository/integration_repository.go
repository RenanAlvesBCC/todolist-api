package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

// IntegrationRepository guarda tokens de integração e execuções da sincronização.
type IntegrationRepository struct {
	db *gorm.DB
}

func NewIntegrationRepository(db *gorm.DB) *IntegrationRepository {
	return &IntegrationRepository{db: db}
}

func (r *IntegrationRepository) CreateToken(t *models.IntegrationToken) error {
	return r.db.Create(t).Error
}

// ListTokens: tokens da oficina (inclusive revogados), mais novos primeiro.
func (r *IntegrationRepository) ListTokens(workspaceID uint) ([]models.IntegrationToken, error) {
	tokens := []models.IntegrationToken{}
	err := r.db.Where("workspace_id = ?", workspaceID).Order("created_at desc").Order("id desc").Find(&tokens).Error
	return tokens, err
}

func (r *IntegrationRepository) FindToken(workspaceID, id uint) (*models.IntegrationToken, error) {
	var t models.IntegrationToken
	if err := r.db.Where("workspace_id = ?", workspaceID).First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *IntegrationRepository) RevokeToken(id uint, at time.Time) error {
	return r.db.Model(&models.IntegrationToken{}).Where("id = ? AND revoked_at IS NULL", id).Update("revoked_at", at).Error
}

// FindActiveTokenByHash devolve nil, nil para hash desconhecido ou token revogado.
func (r *IntegrationRepository) FindActiveTokenByHash(hash string) (*models.IntegrationToken, error) {
	return firstOrNil[models.IntegrationToken](r.db.Where("token_hash = ? AND revoked_at IS NULL", hash))
}

func (r *IntegrationRepository) TouchTokenUsed(id uint, at time.Time) error {
	return r.db.Model(&models.IntegrationToken{}).Where("id = ?", id).Update("last_used_at", at).Error
}

func (r *IntegrationRepository) CreateRun(run *models.SyncRun) error {
	return r.db.Create(run).Error
}

func (r *IntegrationRepository) SaveRun(run *models.SyncRun) error {
	return r.db.Save(run).Error
}

func (r *IntegrationRepository) FindRun(workspaceID, id uint) (*models.SyncRun, error) {
	var run models.SyncRun
	if err := r.db.Where("workspace_id = ?", workspaceID).First(&run, id).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

// ListRuns: execuções mais recentes primeiro.
func (r *IntegrationRepository) ListRuns(workspaceID uint, limit int) ([]models.SyncRun, error) {
	runs := []models.SyncRun{}
	err := r.db.Where("workspace_id = ?", workspaceID).
		Order("started_at desc").Order("id desc").
		Limit(limit).Find(&runs).Error
	return runs, err
}
