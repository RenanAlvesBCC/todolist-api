package services

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
)

type WorkspaceStore interface {
	Create(ws *models.Workspace) error
	FindByOwner(ownerID uint) (*models.Workspace, error)
	FindByID(id uint) (*models.Workspace, error)
	FindByMemberUserID(userID uint) (*models.Workspace, error)
	Update(ws *models.Workspace) error
	AddMember(member *models.WorkspaceMember) error
	FindMember(workspaceID, userID uint) (*models.WorkspaceMember, error)
	UpdateMemberLastSeen(workspaceID, userID uint) error
	RemoveMember(workspaceID, userID uint) error
	ListMembers(workspaceID uint) ([]models.WorkspaceMember, error)
	CreateInvite(invite *models.WorkspaceInvite) error
	FindInviteByCode(code string) (*models.WorkspaceInvite, error)
	ListInvites(workspaceID uint) ([]models.WorkspaceInvite, error)
	MarkInviteUsed(invite *models.WorkspaceInvite) error
	IsMember(workspaceID, userID uint) (bool, error)
	GetMemberRole(workspaceID, userID uint) (models.WorkspaceRole, error)
}

type WorkspaceService struct {
	repo WorkspaceStore
}

func NewWorkspaceService(repo WorkspaceStore) *WorkspaceService {
	return &WorkspaceService{repo: repo}
}

func (s *WorkspaceService) CreateWorkspace(ownerID uint, name, description string) (*models.Workspace, error) {
	if name == "" {
		return nil, errors.New("nome é obrigatório")
	}

	// Uma conta = um workspace
	if existing, err := s.repo.FindByOwner(ownerID); err == nil && existing != nil {
		return nil, errors.New("você já possui um workspace")
	}

	ws := &models.Workspace{Name: name, Description: description, OwnerID: ownerID}
	if err := s.repo.Create(ws); err != nil {
		return nil, err
	}

	// Owner entra automaticamente como membro com role owner
	if err := s.repo.AddMember(&models.WorkspaceMember{
		WorkspaceID: ws.ID,
		UserID:      ownerID,
		Role:        models.RoleOwner,
		JoinedAt:    time.Now(),
	}); err != nil {
		return nil, err
	}

	return ws, nil
}

func (s *WorkspaceService) GetMyWorkspace(userID uint) (*models.Workspace, error) {
	ws, err := s.repo.FindByMemberUserID(userID)
	if err != nil {
		return nil, errors.New("workspace não encontrado")
	}
	return ws, nil
}

func (s *WorkspaceService) UpdateWorkspace(ownerID uint, name, description string) (*models.Workspace, error) {
	if name == "" {
		return nil, errors.New("nome é obrigatório")
	}

	ws, err := s.repo.FindByOwner(ownerID)
	if err != nil {
		return nil, errors.New("workspace não encontrado")
	}

	ws.Name = name
	ws.Description = description
	if err := s.repo.Update(ws); err != nil {
		return nil, err
	}
	return ws, nil
}

func (s *WorkspaceService) GenerateInvite(requesterID uint, role models.WorkspaceRole) (string, error) {
	// Somente owner pode convidar
	ws, err := s.repo.FindByOwner(requesterID)
	if err != nil {
		return "", errors.New("apenas o dono pode gerar convites")
	}

	if role == models.RoleOwner {
		return "", errors.New("não é possível convidar alguém como owner")
	}

	code := uuid.New().String()
	invite := &models.WorkspaceInvite{
		WorkspaceID: ws.ID,
		InvitedBy:   requesterID,
		Code:        code,
		Role:        role,
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.repo.CreateInvite(invite); err != nil {
		return "", err
	}
	return code, nil
}

func (s *WorkspaceService) AcceptInvite(userID uint, code string) (*models.Workspace, error) {
	invite, err := s.repo.FindInviteByCode(code)
	if err != nil {
		return nil, errors.New("convite não encontrado")
	}
	if invite.UsedAt != nil {
		return nil, errors.New("convite já foi utilizado")
	}
	if time.Now().After(invite.ExpiresAt) {
		return nil, errors.New("convite expirado")
	}

	isMember, _ := s.repo.IsMember(invite.WorkspaceID, userID)
	if isMember {
		return nil, errors.New("você já é membro deste workspace")
	}

	if err := s.repo.AddMember(&models.WorkspaceMember{
		WorkspaceID: invite.WorkspaceID,
		UserID:      userID,
		Role:        invite.Role,
		JoinedAt:    time.Now(),
	}); err != nil {
		return nil, err
	}

	now := time.Now()
	invite.UsedAt = &now
	invite.UsedBy = &userID
	if err := s.repo.MarkInviteUsed(invite); err != nil {
		return nil, err
	}

	return s.repo.FindByID(invite.WorkspaceID)
}

func (s *WorkspaceService) ListMembers(requesterID uint) ([]models.WorkspaceMember, error) {
	ws, err := s.repo.FindByMemberUserID(requesterID)
	if err != nil {
		return nil, errors.New("workspace não encontrado")
	}
	return s.repo.ListMembers(ws.ID)
}

func (s *WorkspaceService) ListInvites(ownerID uint) ([]models.WorkspaceInvite, error) {
	ws, err := s.repo.FindByOwner(ownerID)
	if err != nil {
		return nil, errors.New("apenas o dono pode ver os convites")
	}
	return s.repo.ListInvites(ws.ID)
}

func (s *WorkspaceService) RemoveMember(ownerID, targetUserID uint) error {
	ws, err := s.repo.FindByOwner(ownerID)
	if err != nil {
		return errors.New("apenas o dono pode remover membros")
	}

	// Owner não pode ser removido
	if targetUserID == ownerID {
		return errors.New("o dono não pode ser removido do workspace")
	}

	// Verifica se o role do target não é owner (proteção extra)
	role, err := s.repo.GetMemberRole(ws.ID, targetUserID)
	if err != nil {
		return errors.New("membro não encontrado")
	}
	if role == models.RoleOwner {
		return errors.New("o dono não pode ser removido do workspace")
	}

	return s.repo.RemoveMember(ws.ID, targetUserID)
}

func (s *WorkspaceService) UpdateLastSeen(userID uint) {
	ws, err := s.repo.FindByMemberUserID(userID)
	if err != nil {
		return
	}
	// Fire and forget — não bloqueia a requisição
	go s.repo.UpdateMemberLastSeen(ws.ID, userID)
}

func (s *WorkspaceService) GetInvitePreview(code string) (*models.WorkspaceInvite, *models.Workspace, error) {
	invite, err := s.repo.FindInviteByCode(code)
	if err != nil {
		return nil, nil, errors.New("convite não encontrado")
	}
	if invite.UsedAt != nil {
		return nil, nil, errors.New("convite já foi utilizado")
	}
	if time.Now().After(invite.ExpiresAt) {
		return nil, nil, errors.New("convite expirado")
	}

	ws, err := s.repo.FindByID(invite.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}
	return invite, ws, nil
}
