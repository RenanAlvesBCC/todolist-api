package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

type WorkspaceRepository struct {
	db *gorm.DB
}

func NewWorkspaceRepository(db *gorm.DB) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

func (r *WorkspaceRepository) Create(ws *models.Workspace) error {
	return r.db.Create(ws).Error
}

func (r *WorkspaceRepository) FindByOwner(ownerID uint) (*models.Workspace, error) {
	var ws models.Workspace
	if err := r.db.Where("owner_id = ?", ownerID).First(&ws).Error; err != nil {
		return nil, err
	}
	return &ws, nil
}

func (r *WorkspaceRepository) FindByID(id uint) (*models.Workspace, error) {
	var ws models.Workspace
	if err := r.db.Preload("Members").First(&ws, id).Error; err != nil {
		return nil, err
	}
	return &ws, nil
}

func (r *WorkspaceRepository) FindByMemberUserID(userID uint) (*models.Workspace, error) {
	var member models.WorkspaceMember
	if err := r.db.Where("user_id = ?", userID).First(&member).Error; err != nil {
		return nil, err
	}
	return r.FindByID(member.WorkspaceID)
}

func (r *WorkspaceRepository) Update(ws *models.Workspace) error {
	return r.db.Save(ws).Error
}

func (r *WorkspaceRepository) AddMember(member *models.WorkspaceMember) error {
	return r.db.Create(member).Error
}

func (r *WorkspaceRepository) FindMember(workspaceID, userID uint) (*models.WorkspaceMember, error) {
	var member models.WorkspaceMember
	if err := r.db.Where("workspace_id = ? AND user_id = ?", workspaceID, userID).First(&member).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *WorkspaceRepository) UpdateMemberLastSeen(workspaceID, userID uint) error {
	now := time.Now()
	return r.db.Model(&models.WorkspaceMember{}).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Update("last_seen_at", now).Error
}

func (r *WorkspaceRepository) RemoveMember(workspaceID, userID uint) error {
	return r.db.Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Delete(&models.WorkspaceMember{}).Error
}

func (r *WorkspaceRepository) ListMembers(workspaceID uint) ([]models.WorkspaceMember, error) {
	var members []models.WorkspaceMember
	if err := r.db.Where("workspace_id = ?", workspaceID).
		Order("joined_at asc").
		Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

func (r *WorkspaceRepository) CreateInvite(invite *models.WorkspaceInvite) error {
	return r.db.Create(invite).Error
}

func (r *WorkspaceRepository) FindInviteByCode(code string) (*models.WorkspaceInvite, error) {
	var invite models.WorkspaceInvite
	if err := r.db.Where("code = ?", code).First(&invite).Error; err != nil {
		return nil, err
	}
	return &invite, nil
}

func (r *WorkspaceRepository) ListInvites(workspaceID uint) ([]models.WorkspaceInvite, error) {
	var invites []models.WorkspaceInvite
	if err := r.db.Where("workspace_id = ?", workspaceID).
		Order("created_at desc").
		Find(&invites).Error; err != nil {
		return nil, err
	}
	return invites, nil
}

func (r *WorkspaceRepository) MarkInviteUsed(invite *models.WorkspaceInvite) error {
	return r.db.Save(invite).Error
}

func (r *WorkspaceRepository) IsMember(workspaceID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.WorkspaceMember{}).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *WorkspaceRepository) GetMemberRole(workspaceID, userID uint) (models.WorkspaceRole, error) {
	var member models.WorkspaceMember
	if err := r.db.Select("role").
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		First(&member).Error; err != nil {
		return "", err
	}
	return member.Role, nil
}
