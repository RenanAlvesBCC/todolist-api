package services

import (
	"errors"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

var ErrNoWorkspace = errors.New("oficina não encontrada")

// WorkspaceRoleStore é o mínimo para descobrir a oficina e o papel de quem pede.
type WorkspaceRoleStore interface {
	FindByMemberUserID(userID uint) (*models.Workspace, error)
	GetMemberRole(workspaceID, userID uint) (models.WorkspaceRole, error)
}

// memberWorkspace devolve a oficina e o papel do usuário; sem oficina → ErrNoWorkspace.
func memberWorkspace(ws WorkspaceRoleStore, userID uint) (uint, models.WorkspaceRole, error) {
	w, err := ws.FindByMemberUserID(userID)
	if err != nil {
		return 0, "", ErrNoWorkspace
	}
	role, err := ws.GetMemberRole(w.ID, userID)
	if err != nil {
		return 0, "", ErrNoWorkspace
	}
	return w.ID, role, nil
}

// managerWorkspace exige dono ou gerente; o mecânico recebe forbidden.
func managerWorkspace(ws WorkspaceRoleStore, userID uint, forbidden error) (uint, error) {
	wsID, role, err := memberWorkspace(ws, userID)
	if err != nil {
		return 0, err
	}
	if role != models.RoleOwner && role != models.RoleManager {
		return 0, forbidden
	}
	return wsID, nil
}

// DuplicateError indica conflito (409) com um cadastro já existente na oficina.
type DuplicateError struct {
	Err        error
	ExistingID uint
}

func (e *DuplicateError) Error() string { return e.Err.Error() }
func (e *DuplicateError) Unwrap() error { return e.Err }
