package services

import (
	"errors"
	"math"
	"time"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
)

type AuditLogStore interface {
	ListLogs(filter repository.AuditFilter) ([]repository.AuditLogRow, int64, error)
}

type AuditWorkspaceStore interface {
	FindByMemberUserID(userID uint) (*models.Workspace, error)
	GetMemberRole(workspaceID, userID uint) (models.WorkspaceRole, error)
}

type AuditService struct {
	repo    AuditLogStore
	wsStore AuditWorkspaceStore
}

func NewAuditService(repo AuditLogStore, wsStore AuditWorkspaceStore) *AuditService {
	return &AuditService{repo: repo, wsStore: wsStore}
}

type PaginatedAudit struct {
	Logs       []repository.AuditLogRow `json:"logs"`
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	TotalPages int                      `json:"total_pages"`
}

func (s *AuditService) List(userID uint, category string, actorID *uint, dateFrom, dateTo *time.Time, page, limit int) (*PaginatedAudit, error) {
	ws, err := s.wsStore.FindByMemberUserID(userID)
	if err != nil {
		return nil, errors.New("workspace não encontrado")
	}
	role, err := s.wsStore.GetMemberRole(ws.ID, userID)
	if err != nil || role == models.RoleEditor {
		return nil, errors.New("apenas gerentes podem ver a auditoria")
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	rows, total, err := s.repo.ListLogs(repository.AuditFilter{
		Category: category,
		ActorID:  actorID,
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	return &PaginatedAudit{Logs: rows, Total: total, Page: page, TotalPages: totalPages}, nil
}
